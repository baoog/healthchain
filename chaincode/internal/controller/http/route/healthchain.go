package route

import (
	controller "consentcc/internal/controller/http"
	"consentcc/internal/controller/http/middleware"
	"consentcc/internal/infra/server"
	"consentcc/internal/model"
)

type HealthChainRoute struct {
	srv                   server.EchoServer
	healthChainController controller.IHealthChainController
}

func NewHealthChainRoute(
	srv server.EchoServer,
	healthChainController controller.IHealthChainController) *HealthChainRoute {
	e := srv.Echo()

	jwtCfg := middleware.JWTConfig{
		SigningKey: []byte("a-string-secret-at-least-256-bits-long"), // TODO: from env
	}
	gr := e.Group("/healthchain/v1")
	// Here you can add more route handlers to the group 'gr' as needed.
	gr.GET("/check", healthChainController.Get)
	//e.GET("/consents", healthChainController.listConsents)
	//e.GET("/audits", healthChainController.listAudits)

	gr.POST("/bundles", healthChainController.CreateBundle, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Doctor, model.RolesConst.Patient))
	gr.POST("/consents/grant", healthChainController.GrantConsent, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Patient))
	gr.POST("/consents/revoke", healthChainController.RevokeConsent, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Patient))

	gr.POST("/fabric/evaluate", healthChainController.EvaluateFabric, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Doctor, model.RolesConst.Patient))
	gr.GET("/fabric/bundles/:id", healthChainController.ReadBundle, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Doctor, model.RolesConst.Patient))
	gr.GET("/fabric/bundles", healthChainController.GetBundlesByOwner, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Patient))
	gr.GET("/fabric/consents/:bundleId/", healthChainController.ReadConsent, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Doctor, model.RolesConst.Patient))
	gr.GET("/fabric/:owner/bundles", healthChainController.GetBundlesByOwner, middleware.AuthMiddleware(jwtCfg, model.RolesConst.Doctor, model.RolesConst.Patient))

	return &HealthChainRoute{srv: srv, healthChainController: healthChainController}
}
