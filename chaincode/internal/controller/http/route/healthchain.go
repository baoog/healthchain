package route

import (
	"consentcc/config"
	controller "consentcc/internal/controller/http"
	"consentcc/internal/controller/http/middleware"
	"consentcc/internal/infra/client/jwt"
	"consentcc/internal/infra/server"
	"consentcc/internal/model"
	middleware2 "github.com/labstack/echo/v4/middleware"
)

type HealthChainRoute struct {
	srv                    server.EchoServer
	healthChainController  controller.IHealthChainController
	orchestratorController controller.IOrchestratorController
	jwtClaim               jwt.JWTClientProvider
}

func NewHealthChainRoute(
	cfg *config.Config,
	srv server.EchoServer,
	healthChainController controller.IHealthChainController,
	orchestratorController controller.IOrchestratorController,
	jwtClaim jwt.JWTClientProvider,
) *HealthChainRoute {
	e := srv.Echo()
	e.Use(middleware2.CORS())
	e.Use(middleware2.Recover())
	mid := middleware.NewAuthMiddleware(jwtClaim, cfg)
	gr := e.Group("/healthchain/v1")
	// Here you can add more route handlers to the group 'gr' as needed.
	gr.GET("/check", healthChainController.Get)
	//e.GET("/consents", healthChainController.listConsents)
	//e.GET("/audits", healthChainController.listAudits)

	gr.POST("/bundles", healthChainController.CreateBundle, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))
	gr.POST("/consents/grant", healthChainController.GrantConsent, mid.AuthMiddleware(model.RolesConst.Patient))
	gr.POST("/consents/revoke", healthChainController.RevokeConsent, mid.AuthMiddleware(model.RolesConst.Patient))
	gr.POST("/consents/emergency/request", healthChainController.RequestEmergencyAccess, mid.AuthMiddleware(model.RolesConst.Doctor))
	gr.GET("/consents/emergency/requests", healthChainController.GetMyEmergencyAccessRequests, mid.AuthMiddleware(model.RolesConst.Doctor))
	gr.POST("/consents/emergency/request/approve", healthChainController.ApproveEmergencyAccess, mid.AuthMiddleware(model.RolesConst.HoD))
	gr.GET("/consents/emergency/request/pending", healthChainController.GetEmergencyAccessRequestsPending, mid.AuthMiddleware(model.RolesConst.HoD))
	gr.GET("/consents/emergency/request/:requestID", healthChainController.GetBreakGlassRequestByRequestID, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.HoD))
	gr.GET("/consents/emergency/request/:requestId/bundles", healthChainController.GetEmergencyBundlesForDoctor, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.HoD))

	gr.POST("/fabric/evaluate", healthChainController.EvaluateFabric, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))
	//gr.GET("/fabric/bundles/granted", healthChainController.GetBundlesGrantedToDoctorPaged, mid.AuthMiddleware(model.RolesConst.Doctor))
	gr.GET("/fabric/bundles/:id", healthChainController.ReadBundle, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))

	gr.POST("/fabric/get-bundle", healthChainController.ReadBundle, mid.BasicAuthMiddleware())

	gr.GET("/fabric/bundles", healthChainController.GetBundlesByOwner, mid.AuthMiddleware(model.RolesConst.Patient))
	gr.GET("/fabric/consents", healthChainController.ListMyConsents, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))
	gr.GET("/fabric/consents/:bundleId", healthChainController.ReadConsent, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))
	gr.GET("/fabric/:owner/bundles", healthChainController.GetBundlesByOwner, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))

	apiGroup := e.Group("/api/v1")
	apiGroup.POST("/init", orchestratorController.InitLedger, mid.BasicAuthMiddleware())
	apiGroup.GET("/bundles/granted", orchestratorController.GetBundlesGrantedToDoctorPaged, mid.AuthMiddleware(model.RolesConst.Doctor))
	apiGroup.GET("/bundles/:id/content", orchestratorController.GetBundleDetail, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))
	apiGroup.POST("/bundles/content", orchestratorController.ExternalGetBundleDetailFor, mid.BasicAuthMiddleware())

	apiGroup.POST("/user", orchestratorController.RegisterUser)
	apiGroup.POST("/user/login", orchestratorController.LoginUser)
	apiGroup.GET("/user/me", orchestratorController.GetMe, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient, model.RolesConst.HoD))
	apiGroup.GET("/user/doctor/search-by-name", orchestratorController.SearchDoctorByName, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))
	apiGroup.GET("/user/patient/get-by-hash", orchestratorController.GetPatientByHash, mid.AuthMiddleware(model.RolesConst.Doctor, model.RolesConst.Patient))

	return &HealthChainRoute{srv: srv, healthChainController: healthChainController}
}
