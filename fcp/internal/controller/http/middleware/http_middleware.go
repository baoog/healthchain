package middleware

import (
	"fcp/config"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
)

type HTTPMiddleware struct {
	cfg *config.Config
}

func NewHTTPMiddleware(cfg *config.Config) *HTTPMiddleware {
	return &HTTPMiddleware{cfg: cfg}
}

func (m *HTTPMiddleware) AdminKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		want := os.Getenv("FCP_ADMIN_KEY")
		got := c.Request().Header.Get("X-FCP-ADMIN-KEY")
		if want == "" || got != want {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		return next(c)
	}
}
