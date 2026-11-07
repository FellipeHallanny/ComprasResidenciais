package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type cachedUser struct {
	user      *domain.User
	expiresAt time.Time
}

type authService struct {
	userRepo  ports.UserRepository
	jwtSecret []byte
	userCache sync.Map // cache com TTL de 60s para evitar 1 query ao Postgres por request HTTP
}

func NewAuthService(userRepo ports.UserRepository, jwtSecret string) ports.AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *authService) generateToken(user *domain.User) (string, error) {
	claims := AuthClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 dias
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "compras-residenciais",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *authService) Login(ctx context.Context, email, password string) (*domain.LoginResponse, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if cleanEmail == "" || password == "" {
		return nil, errors.New("e-mail e senha são obrigatórios")
	}

	user, err := s.userRepo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return nil, errors.New("credenciais inválidas")
	}

	if user.IsBlocked {
		return nil, errors.New("usuário bloqueado pelo administrador")
	}

	if strings.TrimSpace(user.PasswordHash) == "" {
		return nil, errors.New("este morador ainda não possui senha cadastrada. Solicite ao administrador da residência que cadastre sua senha de acesso")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("credenciais inválidas")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar token de autenticação: %w", err)
	}

	// Armazena no cache para os próximos requests
	s.userCache.Store(user.ID, cachedUser{
		user:      user,
		expiresAt: time.Now().Add(60 * time.Second),
	})

	return &domain.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) SetupInitialAdmin(ctx context.Context, name, email, password string) (*domain.LoginResponse, error) {
	cleanName := strings.TrimSpace(name)
	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	if cleanName == "" || cleanEmail == "" || len(password) < 6 {
		return nil, errors.New("nome, e-mail e senha (mínimo 6 caracteres) são obrigatórios")
	}

	adminCount, err := s.userRepo.CountAdmins(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao verificar administradores existentes: %w", err)
	}

	if adminCount > 0 {
		return nil, errors.New("o administrador inicial já foi configurado no sistema")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("falha ao criptografar senha: %w", err)
	}

	var user *domain.User
	// Verifica se este morador já existe (ex: cadastrado antes da ativação do módulo de segurança)
	existing, err := s.userRepo.GetByEmail(ctx, cleanEmail)
	if err == nil && existing != nil {
		existing.Name = cleanName
		existing.PasswordHash = string(hash)
		existing.Role = domain.UserRoleAdmin
		existing.IsBlocked = false
		existing.UpdatedAt = time.Now()
		if err := s.userRepo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("falha ao promover morador existente para administrador: %w", err)
		}
		user = existing
	} else {
		user = &domain.User{
			ID:           uuid.New(),
			Name:         cleanName,
			Email:        cleanEmail,
			PasswordHash: string(hash),
			Role:         domain.UserRoleAdmin,
			IsBlocked:    false,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("falha ao criar administrador inicial: %w", err)
		}
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar token para administrador: %w", err)
	}

	s.userCache.Store(user.ID, cachedUser{
		user:      user,
		expiresAt: time.Now().Add(60 * time.Second),
	})

	return &domain.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) GetAuthStatus(ctx context.Context) (bool, error) {
	count, err := s.userRepo.CountAdmins(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*domain.User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, errors.New("token inválido ou expirado")
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token inválido")
	}

	// OTIMIZAÇÃO: Cache in-memory com TTL de 60s para poupar 80%+ das requisições SQL
	if val, ok := s.userCache.Load(claims.UserID); ok {
		entry := val.(cachedUser)
		if time.Now().Before(entry.expiresAt) {
			if entry.user.IsBlocked {
				return nil, errors.New("usuário bloqueado pelo administrador")
			}
			return entry.user, nil
		}
		s.userCache.Delete(claims.UserID)
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("usuário associado ao token não encontrado")
	}

	s.userCache.Store(claims.UserID, cachedUser{
		user:      user,
		expiresAt: time.Now().Add(60 * time.Second),
	})

	if user.IsBlocked {
		return nil, errors.New("usuário bloqueado pelo administrador")
	}

	return user, nil
}
