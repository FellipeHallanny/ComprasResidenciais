package http

import (
	"net/http"
	"strings"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/gin-gonic/gin"
)

const CurrentUserContextKey = "current_user"

// AuthMiddleware valida o token JWT contido no header Authorization: Bearer <token>
func AuthMiddleware(authService ports.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Acesso não autorizado: token de autenticação não informado",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Formato de autorização inválido. Utilize 'Bearer <token>'",
			})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		user, err := authService.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.Set(CurrentUserContextKey, user)
		c.Next()
	}
}

// RequireAdminMiddleware assegura que apenas usuários com papel de ADMIN possam prosseguir
func RequireAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Acesso não autorizado: usuário não identificado",
			})
			return
		}

		if user.Role != domain.UserRoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Acesso negado: operação exclusiva para administradores da residência",
			})
			return
		}

		c.Next()
	}
}

// GetCurrentUser recupera o usuário autenticado do contexto do Gin
func GetCurrentUser(c *gin.Context) *domain.User {
	val, exists := c.Get(CurrentUserContextKey)
	if !exists {
		return nil
	}
	user, ok := val.(*domain.User)
	if !ok {
		return nil
	}
	return user
}
