package middleware

import (
	"consentcc/config"
	jwt2 "consentcc/internal/infra/client/jwt"
	"consentcc/internal/model"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"strings"
)

type Middleware struct {
	jwt jwt2.JWTClientProvider
	cfg *config.Config
}

func NewAuthMiddleware(jwt jwt2.JWTClientProvider, cfg *config.Config) *Middleware {
	return &Middleware{jwt: jwt, cfg: cfg}
}

func (m Middleware) AuthMiddleware(roles ...model.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing Authorization header"})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid Authorization header"})
			}

			tokenString := parts[1]
			claim, err := m.jwt.Verify(tokenString)
			if err != nil || claim == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			if claim.UserID == "" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "userId claim missing"})
			}

			if claim.Role == "" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "role claim missing"})
			}

			if claim.NationalIdHash == "" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "id claim missing"})
			}

			for _, role := range roles {
				if model.Role(claim.Role) == role {
					// store in context
					c.Set("userId", claim.UserID)
					c.Set("role", claim.Role)
					c.Set("nationalIdHash", claim.NationalIdHash)
					// set department if doctor
					if claim.Role == string(model.RolesConst.Doctor) {
						c.Set("department", claim.Department)
					}
					return next(c)
				}
			}
			return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient role"})
		}
	}
}

func (m Middleware) BasicAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("X-API-KEY")
			fmt.Println("BasicAuthMiddleware", authHeader, m.cfg.APIKey)
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing Authorization header"})
			}
			if authHeader != m.cfg.APIKey {
				log.Default().Println("Invalid API key:", authHeader, "expected:", m.cfg.APIKey)
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
			}

			return next(c)
		}
	}
}
