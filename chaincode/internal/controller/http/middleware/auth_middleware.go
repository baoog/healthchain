package middleware

import (
	"consentcc/internal/model"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	SigningKey []byte
}

func AuthMiddleware(cfg JWTConfig, roles ...model.Role) echo.MiddlewareFunc {
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
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return cfg.SigningKey, nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token claims"})
			}

			rawUserID, ok := claims["userId"]
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "userId claim missing"})
			}
			userId, _ := rawUserID.(string)
			if userId == "" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "userId empty"})
			}

			rawRole, ok := claims["role"]
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "role claim missing"})
			}

			roleStr, _ := rawRole.(string)
			for _, role := range roles {
				if model.Role(roleStr) == role {
					// store in context
					c.Set("userId", userId)
					c.Set("role", roleStr)

					if roleStr == string(model.RolesConst.Doctor) {
						rawDepartment, ok := claims["department"]
						if ok {
							department, _ := rawDepartment.(string)
							c.Set("department", department)
						}
					}
					return next(c)
				}
			}
			return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient role"})
		}
	}
}
