package route

import (
	"fcp/config"
	"fcp/internal/controller/http"
	"fcp/internal/infra/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type FcpRoute struct {
	sv                   server.EchoServer
	cfg                  *config.Config
	federationController *http.FederationHandler
}

func NewFcpRoute(cfg *config.Config, sv server.EchoServer, federationController *http.FederationHandler) *FcpRoute {
	e := sv.Echo()
	//httpx := middleware2.NewHTTPMiddleware(cfg)
	// Define your routes here, for example:
	e.Use(middleware.Recover())

	v1 := e.Group("/fcp/v1")
	v1.GET("/health", func(c echo.Context) error {
		return c.JSON(200, echo.Map{
			"status": "ok",
		})
	})
	//_ = v1.Group("/admin", httpx.AdminKeyMiddleware)
	v1.POST("/hospitals/register", federationController.Register)
	v1.PUT("/hospitals/:hospitalId/status", federationController.SetStatus)

	// Public resolve endpoint (used by Hospital A orchestrator)
	v1.GET("/hospitals/:hospitalId/resolve", federationController.Resolve)

	// Hospital self-service key rotation (client credentials)
	v1.PUT("/hospitals/:hospitalId/keys/rotate", federationController.RotateKey)

	return &FcpRoute{
		sv:                   sv,
		cfg:                  cfg,
		federationController: federationController,
	}
}
