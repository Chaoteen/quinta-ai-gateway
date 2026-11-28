package model

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"strings"
	"sync"
	"testing"
)

func rbacFixture(t *testing.T) {
	t.Helper()
	var dialect gorm.Dialector = sqlite.Open(":memory:")
	if driver := os.Getenv("RBAC_TEST_DB_DRIVER"); driver != "" {
		dsn := os.Getenv("RBAC_TEST_DB_DSN")
		if os.Getenv("RBAC_ALLOW_DISPOSABLE_DB") != "true" || !strings.Contains(dsn, "security_test") {
			t.Fatal("RBAC external DB tests require an explicitly disposable security_test database")
		}
		switch driver {
		case "mysql":
			dialect = mysql.Open(dsn)
		case "postgres":
			dialect = postgres.Open(dsn)
		default:
			t.Fatal("unsupported RBAC test driver")
		}
	}
	db, err := gorm.Open(dialect, &gorm.Config{})
	require.NoError(t, err)
	old := DB
	DB = db
	t.Cleanup(func() { DB = old; s, _ := db.DB(); _ = s.Close() })
	if os.Getenv("RBAC_TEST_DB_DRIVER") != "" {
		require.NoError(t, db.Migrator().DropTable(&RBACBinding{}, &RBACAudit{}, &ScopedRole{}, &RBACPolicy{}, &User{}, &Tenant{}))
	}
	connection, err := db.DB()
	require.NoError(t, err)
	if os.Getenv("RBAC_TEST_DB_DRIVER") == "" {
		connection.SetMaxOpenConns(1)
	}
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &RBACPolicy{}, &ScopedRole{}, &RBACBinding{}, &RBACAudit{}))
	for _, u := range []User{
		{Id: 1, Username: "root", AffCode: "r", Role: 100, RoleKey: "root", Status: 1, TenantId: 1},
		{Id: 2, Username: "enterprise", AffCode: "e", Role: 10, RoleKey: "enterprise_admin", Status: 1, TenantId: 2},
		{Id: 3, Username: "member", AffCode: "m", Role: 1, RoleKey: "user", Status: 1, TenantId: 2},
		{Id: 4, Username: "foreign", AffCode: "f", Role: 1, RoleKey: "user", Status: 1, TenantId: 3},
		{Id: 5, Username: "peer", AffCode: "p", Role: 10, RoleKey: "enterprise_admin", Status: 1, TenantId: 2},
	} {
		require.NoError(t, db.Create(&u).Error)
	}
	require.NoError(t, db.Create(&Tenant{Id: 2, Name: "fixture-enterprise", Status: 1}).Error)
	require.NoError(t, db.Create(&Tenant{Id: 3, Name: "fixture-foreign", Status: 1}).Error)
}

func TestRBACDelegationAndLiveRevocation(t *testing.T) {
	rbacFixture(t)
	family := common.RoleKeyEnterpriseAdmin
	caps := common.ResourcePermissions(family)
	roleInput := ScopedRoleInput{TenantId: 2, Family: family, Name: "Finance", Active: true, Grants: map[string]common.ResourceGrant{"invoice_requests": caps["invoice_requests"]}}
	_, err := SaveScopedRole(2, roleInput)
	require.Error(t, err, "delegation starts disabled")
	_, err = SaveRBACPolicy(2, family, caps, true, 0)
	require.Error(t, err, "tenant cannot set platform policy")
	policy, err := SaveRBACPolicy(1, family, caps, true, 0)
	require.NoError(t, err)
	role, err := SaveScopedRole(2, roleInput)
	require.NoError(t, err)
	require.Error(t, AssignScopedRole(2, 4, role.Id), "cross tenant")
	require.Error(t, AssignScopedRole(2, 2, role.Id), "self assignment")
	require.Error(t, AssignScopedRole(2, 5, role.Id), "peer admin")
	require.NoError(t, AssignScopedRole(2, 3, role.Id))
	state, err := EffectiveRBAC(3)
	require.NoError(t, err)
	require.True(t, state.Grants["invoice_requests"].Create)
	require.False(t, state.Grants["members"].View)
	require.False(t, state.CanDelegate, "assigned role cannot become role manager")
	allowed, inferenceErr := RBACInferenceAllowed(3)
	require.NoError(t, inferenceErr)
	require.False(t, allowed, "existing inference keys cannot bypass assigned role")
	var user User
	require.NoError(t, DB.First(&user, 3).Error)
	require.EqualValues(t, 1, user.AuthEpoch)
	roleInput.Id = role.Id
	roleInput.Version = role.Version
	roleInput.Grants = map[string]common.ResourceGrant{}
	_, err = SaveScopedRole(2, roleInput)
	require.NoError(t, err)
	state, err = EffectiveRBAC(3)
	require.NoError(t, err)
	require.False(t, state.Grants["invoice_requests"].Create, "no permission cache")
	require.Error(t, RevokeScopedRole(2, 4), "foreign revocation")
	require.NoError(t, RevokeScopedRole(2, 3))
	require.NoError(t, DB.First(&user, 3).Error)
	require.Equal(t, "user", user.RoleKey)
	require.EqualValues(t, 2, user.AuthEpoch)
	caps["invoice_requests"] = common.ResourceGrant{Scope: "tenant"}
	_, err = SaveRBACPolicy(1, family, caps, false, policy.Version)
	require.NoError(t, err)
	_, err = SaveScopedRole(2, ScopedRoleInput{TenantId: 2, Family: family, Name: "Denied", Active: true, Grants: map[string]common.ResourceGrant{}})
	require.Error(t, err)
}

func TestRBACPolicyCeilingCASAndAtomicAudit(t *testing.T) {
	rbacFixture(t)
	family := common.RoleKeyEnterpriseAdmin
	caps := common.ResourcePermissions(family)
	bad := common.ResourcePermissions(family)
	bad["invoice_issuance"] = common.ResourceGrant{View: true, Operate: true, Scope: "tenant"}
	_, err := SaveRBACPolicy(1, family, bad, true, 0)
	require.Error(t, err, "cannot grant platform-only action")
	policy, err := SaveRBACPolicy(1, family, caps, true, 0)
	require.NoError(t, err)
	_, err = SaveRBACPolicy(1, family, caps, true, 0)
	require.Error(t, err, "stale update")
	role, err := SaveScopedRole(2, ScopedRoleInput{TenantId: 2, Family: family, Name: "Member", Active: true, Grants: map[string]common.ResourceGrant{"keys": caps["keys"]}})
	require.NoError(t, err)
	require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("rbac:audit-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "rbac_audits" {
			tx.AddError(errors.New("audit down"))
		}
	}))
	require.Error(t, AssignScopedRole(2, 3, role.Id))
	var user User
	require.NoError(t, DB.First(&user, 3).Error)
	require.Equal(t, "user", user.RoleKey)
	require.Zero(t, user.AuthEpoch)
	require.NoError(t, DB.Callback().Create().Remove("rbac:audit-failure"))
	require.NoError(t, AssignScopedRole(2, 3, role.Id))
	caps["keys"] = common.ResourceGrant{Scope: "tenant"}
	_, err = SaveRBACPolicy(1, family, caps, true, policy.Version)
	require.NoError(t, err)
	state, err := EffectiveRBAC(3)
	require.NoError(t, err)
	require.False(t, state.Grants["keys"].View, "parent ceiling applies live")
}

func TestRBACConcurrentRoleRevision(t *testing.T) {
	rbacFixture(t)
	family := common.RoleKeyEnterpriseAdmin
	caps := common.ResourcePermissions(family)
	_, err := SaveRBACPolicy(1, family, caps, true, 0)
	require.NoError(t, err)
	role, err := SaveScopedRole(2, ScopedRoleInput{TenantId: 2, Family: family, Name: "Versioned", Active: true, Grants: map[string]common.ResourceGrant{}})
	require.NoError(t, err)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, actor := range []int{2, 5} {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			_, err := SaveScopedRole(id, ScopedRoleInput{Id: role.Id, TenantId: 2, Family: family, Name: "Changed", Active: true, Version: role.Version, Grants: map[string]common.ResourceGrant{}})
			results <- err
		}(actor)
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrRBACConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, conflicts)
}
