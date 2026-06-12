package router

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/controller"
	"github.com/Chaoteen/quinta-ai-gateway/middleware"

	// Import oauth package to register providers via init()
	_ "github.com/Chaoteen/quinta-ai-gateway/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.RouteTag("api"))
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.RootAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		//apiRouter.GET("/midjourney", controller.GetMidjourney)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.TryUserAuth(), controller.GetPricing)
		perfMetricsRoute := apiRouter.Group("/perf-metrics")
		perfMetricsRoute.Use(middleware.TryUserAuth())
		{
			perfMetricsRoute.GET("/summary", controller.GetPerfMetricsSummary)
			perfMetricsRoute.GET("", controller.GetPerfMetrics)
		}
		apiRouter.GET("/rankings", controller.GetRankings)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), controller.ResetPassword)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), controller.GenerateOAuthCode)
		apiRouter.POST("/oauth/email/bind", middleware.CriticalRateLimit(), controller.EmailBind)
		// Non-standard OAuth (WeChat, Telegram) - keep original routes
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), controller.WeChatAuth)
		apiRouter.POST("/oauth/wechat/bind", middleware.CriticalRateLimit(), controller.WeChatBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimit(), controller.TelegramLogin)
		apiRouter.GET("/oauth/telegram/bind", middleware.CriticalRateLimit(), controller.TelegramBind)
		// Standard OAuth providers (GitHub, Discord, OIDC, LinuxDO) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), controller.HandleOAuth)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), controller.GetRatioConfig)

		apiRouter.POST("/stripe/webhook", controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", controller.CreemWebhook)
		apiRouter.POST("/waffo/webhook", controller.WaffoWebhook)
		//apiRouter.POST("/waffo-pancake/webhook", controller.WaffoPancakeWebhook)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		billingPortalRoute := apiRouter.Group("/billing")
		billingPortalRoute.Use(middleware.UserAuth())
		{
			billingPortalRoute.GET("/summary", controller.GetBillingPortalSummary)
			billingPortalRoute.GET("/payments", controller.GetBillingPortalPayments)
			billingPortalRoute.GET("/usages", controller.GetBillingPortalUsages)
			billingPortalRoute.GET("/records", controller.GetBillingPortalRecords)
			billingPortalRoute.GET("/subscriptions", controller.GetBillingPortalSubscriptions)
		}

		financeReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance, common.RoleKeyAuditor)
		channelReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOps, common.RoleKeyAuditor)
		subscriptionReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOrganizationAdmin, common.RoleKeyFinance, common.RoleKeyAuditor)
		subscriptionWriteAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin)
		catalogReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOps, common.RoleKeyAuditor)
		billingReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOrganizationAdmin, common.RoleKeyFinance, common.RoleKeyAuditor)
		revenueShareReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance, common.RoleKeyAuditor)
		revenueShareWriteAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin)
		paymentReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance)
		paymentReviewAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance)
		voucherAdminAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin)
		voucherRedemptionReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance)
		financeConsoleAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance)
		invoiceAdminAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance)
		userReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOrganizationAdmin)
		operationalFinanceReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOrganizationAdmin, common.RoleKeyFinance, common.RoleKeyAuditor)
		operationalOpsReadAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOrganizationAdmin, common.RoleKeyOps, common.RoleKeyAuditor)
		channelExecuteAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOps)
		channelBalanceExecuteAuth := middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyOps, common.RoleKeyFinance)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), controller.Verify2FALogin)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", controller.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", controller.PasskeyDelete)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.GET("/topup/info", controller.GetTopUpInfo)
				selfRoute.GET("/topup/self", controller.GetUserTopUps)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), controller.TopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.RequestCreemPay)
				selfRoute.POST("/waffo/amount", controller.RequestWaffoAmount)
				selfRoute.POST("/waffo/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPay)
				//selfRoute.POST("/waffo-pancake/amount", controller.RequestWaffoPancakeAmount)
				//selfRoute.POST("/waffo-pancake/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPancakePay)
				selfRoute.POST("/aff_transfer", controller.TransferAffQuota)
				selfRoute.PUT("/setting", controller.UpdateUserSetting)

				// 2FA routes
				selfRoute.GET("/2fa/status", controller.Get2FAStatus)
				selfRoute.POST("/2fa/setup", controller.Setup2FA)
				selfRoute.POST("/2fa/enable", controller.Enable2FA)
				selfRoute.POST("/2fa/disable", controller.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", controller.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", controller.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), controller.DoCheckin)

				// Custom OAuth bindings
				selfRoute.GET("/oauth/bindings", controller.GetUserOAuthBindings)
				selfRoute.DELETE("/oauth/bindings/:provider_id", controller.UnbindCustomOAuth)
			}

			userRoute.GET("/topup", operationalFinanceReadAuth, controller.GetAllTopUps)
			userRoute.GET("/", userReadAuth, controller.GetAllUsers)
			userRoute.GET("/search", userReadAuth, controller.SearchUsers)
			userRoute.GET("/:id", userReadAuth, controller.GetUser)

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
			{
				adminRoute.POST("/topup/complete", controller.AdminCompleteTopUp)
				adminRoute.GET("/:id/oauth/bindings", controller.GetUserOAuthBindingsByAdmin)
				adminRoute.DELETE("/:id/oauth/bindings/:provider_id", controller.UnbindCustomOAuthByAdmin)
				adminRoute.DELETE("/:id/bindings/:binding_type", controller.AdminClearUserBinding)
				adminRoute.POST("/", controller.CreateUser)
				adminRoute.POST("/manage", controller.ManageUser)
				adminRoute.PUT("/", controller.UpdateUser)
				adminRoute.DELETE("/:id", controller.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", controller.AdminResetPasskey)

				// Admin 2FA routes
				adminRoute.GET("/2fa/stats", middleware.RootAuth(), controller.Admin2FAStats)
				adminRoute.DELETE("/:id/2fa", controller.AdminDisable2FA)
			}
		}

		// Subscription billing (plans, purchase, admin management)
		subscriptionRoute := apiRouter.Group("/subscription")
		subscriptionRoute.Use(middleware.UserAuth())
		{
			subscriptionRoute.GET("/plans", controller.GetSubscriptionPlans)
			subscriptionRoute.GET("/self", controller.GetSubscriptionSelf)
			subscriptionRoute.PUT("/self/preference", controller.UpdateSubscriptionPreference)
			subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)
			subscriptionRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestStripePay)
			subscriptionRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestCreemPay)
		}
		subscriptionAdminRoute := apiRouter.Group("/subscription/admin")
		{
			subscriptionAdminRoute.GET("/plans", billingReadAuth, controller.AdminListSubscriptionPlans)
			subscriptionAdminRoute.POST("/plans", middleware.RootAuth(), controller.AdminCreateSubscriptionPlan)
			subscriptionAdminRoute.PUT("/plans/:id", middleware.RootAuth(), controller.AdminUpdateSubscriptionPlan)
			subscriptionAdminRoute.PATCH("/plans/:id", middleware.RootAuth(), controller.AdminUpdateSubscriptionPlanStatus)
			subscriptionAdminRoute.POST("/bind", subscriptionWriteAuth, controller.AdminBindSubscription)

			// User subscription management (admin)
			subscriptionAdminRoute.GET("/user-subscriptions", subscriptionReadAuth, controller.AdminListAllUserSubscriptions)
			subscriptionAdminRoute.GET("/users/:id/subscriptions", subscriptionReadAuth, controller.AdminListUserSubscriptions)
			subscriptionAdminRoute.POST("/users/:id/subscriptions", subscriptionWriteAuth, controller.AdminCreateUserSubscription)
			subscriptionAdminRoute.PATCH("/user-subscriptions/:id/cancel", subscriptionWriteAuth, controller.AdminCancelUserSubscription)
			subscriptionAdminRoute.PATCH("/user-subscriptions/:id/suspend", subscriptionWriteAuth, controller.AdminSuspendUserSubscription)
			subscriptionAdminRoute.PATCH("/user-subscriptions/:id/renew", subscriptionWriteAuth, controller.AdminRenewUserSubscription)
			subscriptionAdminRoute.POST("/user_subscriptions/:id/invalidate", subscriptionWriteAuth, controller.AdminInvalidateUserSubscription)
			subscriptionAdminRoute.DELETE("/user_subscriptions/:id", subscriptionWriteAuth, controller.AdminDeleteUserSubscription)
		}

		// Subscription payment callbacks (no auth)
		apiRouter.POST("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/return", controller.SubscriptionEpayReturn)
		apiRouter.POST("/subscription/epay/return", controller.SubscriptionEpayReturn)

		revenueShareRoute := apiRouter.Group("/revenue-share")
		{
			revenueShareRoute.POST("/rules", revenueShareWriteAuth, controller.CreateRevenueShareRule)
			revenueShareRoute.GET("/rules", revenueShareReadAuth, controller.ListRevenueShareRules)
			revenueShareRoute.PUT("/rules/:id", revenueShareWriteAuth, controller.UpdateRevenueShareRule)
			revenueShareRoute.POST("/rules/:id/enable", revenueShareWriteAuth, controller.EnableRevenueShareRule)
			revenueShareRoute.POST("/rules/:id/disable", revenueShareWriteAuth, controller.DisableRevenueShareRule)
			revenueShareRoute.GET("/records", revenueShareReadAuth, controller.ListRevenueShareRecords)
		}
		paymentRoute := apiRouter.Group("/payment")
		paymentRoute.Use(middleware.UserAuth())
		{
			paymentRoute.POST("/orders", middleware.CriticalRateLimit(), controller.CreatePaymentOrder)
			paymentRoute.GET("/orders", controller.ListUserPaymentOrders)
			paymentRoute.GET("/orders/:id", controller.GetUserPaymentOrder)
			paymentRoute.POST("/bank-transfer", middleware.CriticalRateLimit(), controller.CreateBankTransferRecord)
		}
		voucherRoute := apiRouter.Group("/vouchers")
		voucherRoute.Use(middleware.UserAuth())
		{
			voucherRoute.POST("/redeem", middleware.CriticalRateLimit(), controller.RedeemVoucher)
			voucherRoute.GET("/history", controller.ListVoucherHistory)
		}
		invoiceRoute := apiRouter.Group("/invoices")
		invoiceRoute.Use(middleware.UserAuth())
		{
			invoiceRoute.POST("/profiles", controller.CreateInvoiceProfile)
			invoiceRoute.GET("/profiles", controller.ListInvoiceProfiles)
			invoiceRoute.POST("/profiles/:id/disable", controller.DisableInvoiceProfile)
			invoiceRoute.POST("/applications", controller.CreateInvoiceApplication)
			invoiceRoute.GET("/applications", controller.ListInvoiceApplications)
			invoiceRoute.GET("/files", controller.ListInvoiceFiles)
		}
		adminPaymentRoute := apiRouter.Group("/admin/payment")
		{
			adminPaymentRoute.GET("/orders", paymentReadAuth, controller.AdminListPaymentOrders)
			adminPaymentRoute.GET("/callback-logs", paymentReadAuth, controller.AdminListPaymentCallbackLogs)
			adminPaymentRoute.GET("/bank-transfers", paymentReadAuth, controller.AdminListBankTransfers)
			adminPaymentRoute.POST("/bank-transfers/:id/review", paymentReviewAuth, controller.AdminReviewBankTransfer)
		}
		adminVoucherRoute := apiRouter.Group("/admin/vouchers")
		{
			adminVoucherRoute.POST("/batches", voucherAdminAuth, controller.AdminCreateVoucherBatch)
			adminVoucherRoute.GET("/batches", voucherAdminAuth, controller.AdminListVoucherBatches)
			adminVoucherRoute.GET("", voucherAdminAuth, controller.AdminListVouchers)
			adminVoucherRoute.GET("/redemptions", voucherRedemptionReadAuth, controller.AdminListVoucherRedemptions)
			adminVoucherRoute.POST("/:id/disable", voucherAdminAuth, controller.AdminDisableVoucher)
		}
		adminVoucherBatchRoute := apiRouter.Group("/admin/voucher-batches")
		{
			adminVoucherBatchRoute.POST("/:id/generate", voucherAdminAuth, controller.AdminGenerateVouchers)
			adminVoucherBatchRoute.POST("/:id/disable", voucherAdminAuth, controller.AdminDisableVoucherBatch)
		}
		adminFinanceRoute := apiRouter.Group("/admin/finance")
		{
			adminFinanceRoute.GET("/summary", financeConsoleAuth, controller.GetFinanceSummary)
			adminFinanceRoute.GET("/top-tenants", financeConsoleAuth, controller.GetFinanceTopTenants)
			adminFinanceRoute.GET("/top-models", financeConsoleAuth, controller.GetFinanceTopModels)
			adminFinanceRoute.GET("/top-providers", financeConsoleAuth, controller.GetFinanceTopProviders)
			adminFinanceRoute.GET("/top-channels", financeConsoleAuth, controller.GetFinanceTopChannels)
			adminFinanceRoute.GET("/recent-payments", financeConsoleAuth, controller.GetFinanceRecentPayments)
			adminFinanceRoute.GET("/recent-redemptions", financeConsoleAuth, controller.GetFinanceRecentRedemptions)
			adminFinanceRoute.GET("/recent-subscriptions", financeConsoleAuth, controller.GetFinanceRecentSubscriptions)
			adminFinanceRoute.GET("/recent-billing", financeConsoleAuth, controller.GetFinanceRecentBilling)
		}
		adminInvoiceRoute := apiRouter.Group("/admin/invoices")
		{
			adminInvoiceRoute.GET("/applications", invoiceAdminAuth, controller.AdminListInvoiceApplications)
			adminInvoiceRoute.POST("/applications/:id/review", invoiceAdminAuth, controller.AdminReviewInvoiceApplication)
			adminInvoiceRoute.POST("/applications/:id/issue", invoiceAdminAuth, controller.AdminIssueInvoice)
			adminInvoiceRoute.GET("/profiles", invoiceAdminAuth, controller.AdminListInvoiceProfiles)
			adminInvoiceRoute.POST("/profiles", invoiceAdminAuth, controller.AdminCreateInvoiceProfile)
			adminInvoiceRoute.GET("/files", invoiceAdminAuth, controller.AdminListInvoiceFiles)
		}
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
			optionRoute.GET("/channel_affinity_cache", controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", controller.ClearChannelAffinityCache)
			optionRoute.POST("/rest_model_ratio", controller.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", controller.MigrateConsoleSetting) // 用于迁移检测的旧键，下个版本会删除
		}

		// Custom OAuth provider management (root only)
		customOAuthRoute := apiRouter.Group("/custom-oauth-provider")
		customOAuthRoute.Use(middleware.RootAuth())
		{
			customOAuthRoute.POST("/discovery", controller.FetchCustomOAuthDiscovery)
			customOAuthRoute.GET("/", controller.GetCustomOAuthProviders)
			customOAuthRoute.GET("/:id", controller.GetCustomOAuthProvider)
			customOAuthRoute.POST("/", controller.CreateCustomOAuthProvider)
			customOAuthRoute.PUT("/:id", controller.UpdateCustomOAuthProvider)
			customOAuthRoute.DELETE("/:id", controller.DeleteCustomOAuthProvider)
		}
		performanceRoute := apiRouter.Group("/performance")
		performanceRoute.Use(middleware.RootAuth())
		{
			performanceRoute.GET("/stats", controller.GetPerformanceStats)
			performanceRoute.DELETE("/disk_cache", controller.ClearDiskCache)
			performanceRoute.POST("/reset_stats", controller.ResetPerformanceStats)
			performanceRoute.POST("/gc", controller.ForceGC)
			performanceRoute.GET("/logs", controller.GetLogFiles)
			performanceRoute.DELETE("/logs", controller.CleanupLogFiles)
		}
		adminConsoleRoute := apiRouter.Group("/admin_console")
		adminConsoleRoute.Use(middleware.RootAuth())
		{
			adminConsoleRoute.GET("/tenants", controller.GetReadonlyTenants)
			adminConsoleRoute.POST("/tenants", controller.CreateAdminConsoleTenant)
			adminConsoleRoute.PUT("/tenants/:id", controller.UpdateAdminConsoleTenant)
			adminConsoleRoute.PATCH("/tenants/:id/status", controller.UpdateAdminConsoleTenantStatus)
			adminConsoleRoute.GET("/organizations", controller.GetReadonlyOrganizations)
			adminConsoleRoute.POST("/organizations", controller.CreateAdminConsoleOrganization)
			adminConsoleRoute.PUT("/organizations/:id", controller.UpdateAdminConsoleOrganization)
			adminConsoleRoute.PATCH("/organizations/:id/status", controller.UpdateAdminConsoleOrganizationStatus)
			adminConsoleRoute.GET("/departments", controller.GetReadonlyDepartments)
			adminConsoleRoute.POST("/departments", controller.CreateAdminConsoleDepartment)
			adminConsoleRoute.PUT("/departments/:id", controller.UpdateAdminConsoleDepartment)
			adminConsoleRoute.PATCH("/departments/:id/status", controller.UpdateAdminConsoleDepartmentStatus)
			adminConsoleRoute.GET("/distribution_channels", controller.GetReadonlyDistributionChannels)
			adminConsoleRoute.POST("/distribution_channels", controller.CreateAdminConsoleDistributionChannel)
			adminConsoleRoute.PUT("/distribution_channels/:id", controller.UpdateAdminConsoleDistributionChannel)
			adminConsoleRoute.PATCH("/distribution_channels/:id/status", controller.UpdateAdminConsoleDistributionChannelStatus)
		}
		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		ratioSyncRoute.Use(middleware.RootAuth())
		{
			ratioSyncRoute.GET("/channels", controller.GetSyncableChannels)
			ratioSyncRoute.POST("/fetch", controller.FetchUpstreamRatios)
		}
		channelRoute := apiRouter.Group("/channel")
		{
			channelRoute.GET("/", channelReadAuth, controller.GetAllChannels)
			channelRoute.GET("/search", channelReadAuth, controller.SearchChannels)
			channelRoute.GET("/models", catalogReadAuth, controller.ChannelListModels)
			channelRoute.GET("/models_enabled", channelReadAuth, controller.EnabledListModels)
			channelRoute.GET("/:id", channelReadAuth, controller.GetChannel)
			channelRoute.POST("/:id/key", middleware.RootAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.SecureVerificationRequired(), controller.GetChannelKey)
			channelRoute.GET("/test", middleware.AdminAuth(), controller.TestAllChannels)
			channelRoute.GET("/test/:id", channelExecuteAuth, controller.TestChannel)
			channelRoute.GET("/update_balance", middleware.AdminAuth(), controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", channelBalanceExecuteAuth, controller.UpdateChannelBalance)
			channelRoute.POST("/", middleware.AdminAuth(), controller.AddChannel)
			channelRoute.PUT("/", middleware.AdminAuth(), controller.UpdateChannel)
			channelRoute.DELETE("/disabled", middleware.AdminAuth(), controller.DeleteDisabledChannel)
			channelRoute.POST("/tag/disabled", middleware.AdminAuth(), controller.DisableTagChannels)
			channelRoute.POST("/tag/enabled", middleware.AdminAuth(), controller.EnableTagChannels)
			channelRoute.PUT("/tag", middleware.AdminAuth(), controller.EditTagChannels)
			channelRoute.DELETE("/:id", middleware.AdminAuth(), controller.DeleteChannel)
			channelRoute.POST("/batch", middleware.AdminAuth(), controller.DeleteChannelBatch)
			channelRoute.POST("/fix", middleware.RootAuth(), controller.FixChannelsAbilities)
			channelRoute.GET("/fetch_models/:id", channelExecuteAuth, controller.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", middleware.RootAuth(), controller.FetchModels)
			channelRoute.POST("/codex/oauth/start", middleware.AdminAuth(), controller.StartCodexOAuth)
			channelRoute.POST("/codex/oauth/complete", middleware.AdminAuth(), controller.CompleteCodexOAuth)
			channelRoute.POST("/:id/codex/oauth/start", middleware.AdminAuth(), controller.StartCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/oauth/complete", middleware.AdminAuth(), controller.CompleteCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/refresh", middleware.AdminAuth(), controller.RefreshCodexChannelCredential)
			channelRoute.GET("/:id/codex/usage", middleware.AdminAuth(), controller.GetCodexChannelUsage)
			channelRoute.POST("/ollama/pull", middleware.AdminAuth(), controller.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", middleware.AdminAuth(), controller.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", middleware.AdminAuth(), controller.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", channelExecuteAuth, controller.OllamaVersion)
			channelRoute.POST("/batch/tag", middleware.AdminAuth(), controller.BatchSetChannelTag)
			channelRoute.GET("/tag/models", channelReadAuth, controller.GetTagModels)
			channelRoute.POST("/copy/:id", middleware.AdminAuth(), controller.CopyChannel)
			channelRoute.POST("/multi_key/manage", middleware.AdminAuth(), controller.ManageMultiKeys)
			channelRoute.POST("/upstream_updates/apply", middleware.AdminAuth(), controller.ApplyChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/apply_all", middleware.AdminAuth(), controller.ApplyAllChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect", middleware.AdminAuth(), controller.DetectChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect_all", middleware.AdminAuth(), controller.DetectAllChannelUpstreamModelUpdates)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", middleware.SearchRateLimit(), controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/:id/key", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKey)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
			tokenRoute.POST("/batch", controller.DeleteTokenBatch)
			tokenRoute.POST("/batch/keys", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKeysBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		{
			redemptionRoute.GET("/", operationalFinanceReadAuth, controller.GetAllRedemptions)
			redemptionRoute.GET("/search", financeReadAuth, controller.SearchRedemptions)
			redemptionRoute.GET("/:id", financeReadAuth, controller.GetRedemption)
			redemptionRoute.POST("/", middleware.AdminAuth(), controller.AddRedemption)
			redemptionRoute.PUT("/", middleware.AdminAuth(), controller.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", middleware.AdminAuth(), controller.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", middleware.AdminAuth(), controller.DeleteRedemption)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", operationalFinanceReadAuth, controller.GetAllLogs)
		logRoute.DELETE("/", middleware.RootAuth(), controller.DeleteHistoryLogs)
		logRoute.GET("/stat", operationalFinanceReadAuth, controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/channel_affinity_usage_cache", middleware.RootAuth(), controller.GetChannelAffinityUsageCacheStats)
		logRoute.GET("/search", middleware.AdminAuth(), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.RootAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/users", middleware.RootAuth(), controller.GetQuotaDatesByUser)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AdminAuth())
		{
			groupRoute.GET("/", controller.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.AdminAuth())
		{
			prefillGroupRoute.GET("/", controller.GetPrefillGroups)
			prefillGroupRoute.POST("/", middleware.RootAuth(), controller.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", middleware.RootAuth(), controller.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", middleware.RootAuth(), controller.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), controller.GetUserMidjourney)
		mjRoute.GET("/", operationalOpsReadAuth, controller.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), controller.GetUserTask)
			taskRoute.GET("/", operationalOpsReadAuth, controller.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		{
			vendorRoute.GET("/", catalogReadAuth, controller.GetAllVendors)
			vendorRoute.GET("/search", catalogReadAuth, controller.SearchVendors)
			vendorRoute.GET("/:id", catalogReadAuth, controller.GetVendorMeta)
			vendorRoute.POST("/", middleware.RootAuth(), controller.CreateVendorMeta)
			vendorRoute.PUT("/", middleware.RootAuth(), controller.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", middleware.RootAuth(), controller.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		{
			modelsRoute.GET("/sync_upstream/preview", middleware.AdminAuth(), controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", middleware.RootAuth(), controller.SyncUpstreamModels)
			modelsRoute.GET("/missing", catalogReadAuth, controller.GetMissingModels)
			modelsRoute.GET("/", catalogReadAuth, controller.GetAllModelsMeta)
			modelsRoute.GET("/search", catalogReadAuth, controller.SearchModelsMeta)
			modelsRoute.GET("/:id", catalogReadAuth, controller.GetModelMeta)
			modelsRoute.POST("/", middleware.RootAuth(), controller.CreateModelMeta)
			modelsRoute.PUT("/", middleware.RootAuth(), controller.UpdateModelMeta)
			modelsRoute.DELETE("/:id", middleware.RootAuth(), controller.DeleteModelMeta)
		}

		// Deployments (model deployment management)
		deploymentsRoute := apiRouter.Group("/deployments")
		deploymentsRoute.Use(middleware.RootAuth())
		{
			deploymentsRoute.GET("/settings", controller.GetModelDeploymentSettings)
			deploymentsRoute.POST("/settings/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/", controller.GetAllDeployments)
			deploymentsRoute.GET("/search", controller.SearchDeployments)
			deploymentsRoute.POST("/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/hardware-types", controller.GetHardwareTypes)
			deploymentsRoute.GET("/locations", controller.GetLocations)
			deploymentsRoute.GET("/available-replicas", controller.GetAvailableReplicas)
			deploymentsRoute.POST("/price-estimation", controller.GetPriceEstimation)
			deploymentsRoute.GET("/check-name", controller.CheckClusterNameAvailability)
			deploymentsRoute.POST("/", controller.CreateDeployment)

			deploymentsRoute.GET("/:id", controller.GetDeployment)
			deploymentsRoute.GET("/:id/logs", controller.GetDeploymentLogs)
			deploymentsRoute.GET("/:id/containers", controller.ListDeploymentContainers)
			deploymentsRoute.GET("/:id/containers/:container_id", controller.GetContainerDetails)
			deploymentsRoute.PUT("/:id", controller.UpdateDeployment)
			deploymentsRoute.PUT("/:id/name", controller.UpdateDeploymentName)
			deploymentsRoute.POST("/:id/extend", controller.ExtendDeployment)
			deploymentsRoute.DELETE("/:id", controller.DeleteDeployment)
		}
	}
}
