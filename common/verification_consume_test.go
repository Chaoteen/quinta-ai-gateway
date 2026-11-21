package common

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRecoveryCodeIsConsumedOnlyOnce(t *testing.T) {
	key := "fixture-consumption"
	RegisterVerificationCodeWithKey(key, "fixture", PasswordResetPurpose)
	if ConsumeVerificationCode(key, "wrong", PasswordResetPurpose) {
		t.Fatal("wrong code accepted")
	}
	var n atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ConsumeVerificationCode(key, "fixture", PasswordResetPurpose) {
				n.Add(1)
			}
		}()
	}
	wg.Wait()
	if n.Load() != 1 {
		t.Fatalf("expected one consumer, got %d", n.Load())
	}
	RegisterVerificationCodeWithKey(key, "replacement", PasswordResetPurpose)
	if !ConsumeVerificationCode(key, "replacement", PasswordResetPurpose) {
		t.Fatal("replacement removed")
	}
	for _, stamp := range []time.Time{time.Now().Add(-time.Hour), time.Now().Add(time.Hour)} {
		verificationMutex.Lock()
		verificationMap[PasswordResetPurpose+key] = verificationValue{code: "fixture", time: stamp}
		verificationMutex.Unlock()
		if ConsumeVerificationCode(key, "fixture", PasswordResetPurpose) {
			t.Fatal("invalid timestamp accepted")
		}
	}
	DeleteKey(key, PasswordResetPurpose)
}
