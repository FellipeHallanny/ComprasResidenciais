package http

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
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
		user, err := authService.ValidateToken(c.Request.Context(), tokenStr)
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

// SecurityHeadersMiddleware adiciona headers de proteção contra clickjacking, MIME sniffing e XSS
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob: *; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		// Se estiver em conexão TLS/HTTPS
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// CORSMiddleware implementa política de CORS estrita sem wildcards conflitantes
func CORSMiddleware() gin.HandlerFunc {
	allowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv != "" {
		for _, o := range strings.Split(allowedOriginsEnv, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}
	if len(allowedOrigins) == 0 {
		// Origens de desenvolvimento locais padrão
		allowedOrigins = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:3000",
			"http://localhost:8080",
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		isAllowed := false

		if origin != "" {
			for _, o := range allowedOrigins {
				if strings.EqualFold(o, origin) || o == "*" {
					isAllowed = true
					break
				}
			}
		}

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// --- Rate Limiting In-Memory com golang.org/x/time/rate ---

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	r       rate.Limit
	b       int
}

// NewIPRateLimiter cria um limitador por IP
// r: taxa de requisições por segundo permitidas
// b: burst (capacidade máxima do bucket)
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		r:       r,
		b:       b,
	}

	// Limpeza periódica de IPs inativos a cada 5 minutos
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.mu.Lock()
			for ip, client := range limiter.clients {
				if time.Since(client.lastSeen) > 10*time.Minute {
					delete(limiter.clients, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	client, exists := i.clients[ip]
	if !exists {
		l := rate.NewLimiter(i.r, i.b)
		i.clients[ip] = &clientLimiter{limiter: l, lastSeen: time.Now()}
		return l
	}

	client.lastSeen = time.Now()
	return client.limiter
}

// RateLimitMiddleware cria middleware para limitar requisições de acordo com a taxa informada
func RateLimitMiddleware(limiter *IPRateLimiter, actionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.getLimiter(ip)
		if !l.Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("Muitas requisições para '%s'. Por favor, aguarde alguns instantes antes de tentar novamente.", actionName),
			})
			return
		}
		c.Next()
	}
}
