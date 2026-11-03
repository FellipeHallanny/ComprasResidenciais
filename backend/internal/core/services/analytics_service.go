package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type analyticsService struct {
	purchaseRepo ports.PurchaseRepository
	productRepo  ports.CanonicalProductRepository
	embedder     ports.EmbeddingProvider
}

// NewAnalyticsService cria instância do serviço de analytics com suporte a pgvector
func NewAnalyticsService(
	purchaseRepo ports.PurchaseRepository,
	productRepo ports.CanonicalProductRepository,
	embedder ports.EmbeddingProvider,
) ports.AnalyticsService {
	return &analyticsService{
		purchaseRepo: purchaseRepo,
		productRepo:  productRepo,
		embedder:     embedder,
	}
}

func (s *analyticsService) GetPriceComparison(ctx context.Context, canonicalProductID uuid.UUID) (*domain.PriceComparisonView, error) {
	return s.purchaseRepo.GetPriceComparison(ctx, canonicalProductID)
}

func (s *analyticsService) GetMonthlyOscillation(ctx context.Context, canonicalProductID uuid.UUID) (*domain.MonthlyPriceOscillation, error) {
	return s.purchaseRepo.GetMonthlyOscillation(ctx, canonicalProductID)
}

func (s *analyticsService) GetGlobalSummary(ctx context.Context, month time.Time) (*domain.GlobalSummaryView, error) {
	return s.purchaseRepo.GetGlobalSummary(ctx, month)
}

func (s *analyticsService) SearchProducts(ctx context.Context, query string) ([]domain.CanonicalProduct, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return s.productRepo.ListAll(ctx)
	}

	// 1. Tenta busca semântica vetorial via pgvector
	if s.embedder != nil {
		vec, err := s.embedder.GenerateEmbedding(ctx, trimmed)
		if err == nil {
			semanticList, err := s.productRepo.SearchSemantic(ctx, vec, 10)
			if err == nil && len(semanticList) > 0 {
				return semanticList, nil
			}
		}
	}

	// 2. Fallback para busca textual ILIKE
	return s.productRepo.SearchByName(ctx, trimmed)
}

// --- UserService Implementation ---

type userService struct {
	userRepo ports.UserRepository
}

// NewUserService cria instância do serviço de usuários
func NewUserService(userRepo ports.UserRepository) ports.UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) CreateUser(ctx context.Context, name, email, password, role string) (*domain.User, error) {
	if role == "" {
		role = domain.UserRoleMember
	}
	var passwordHash string
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = string(hash)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		IsBlocked:    false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.userRepo.ListAll(ctx)
}

func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, name, email, role string, isBlocked bool, newPassword string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Name = name
	user.Email = email
	if role != "" {
		user.Role = role
	}
	user.IsBlocked = isBlocked
	if newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = string(hash)
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) DeleteUser(ctx context.Context, id uuid.UUID) (*ports.DeleteUserResult, error) {
	// 1. Verifica se o usuário existe
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// 2. Verifica se existem compras vinculadas ao usuário
	purchaseCount, err := s.userRepo.CountPurchasesByUserID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check user purchases: %w", err)
	}

	// Se possui histórico de compras, bloqueia o usuário mantendo o histórico
	if purchaseCount > 0 {
		if err := s.userRepo.Block(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to block user: %w", err)
		}
		return &ports.DeleteUserResult{
			Action:  ports.UserActionBlocked,
			Message: fmt.Sprintf("Usuário '%s' possui %d compra(s) associada(s). O usuário foi bloqueado para preservar o histórico financeiro.", user.Name, purchaseCount),
			UserID:  id,
		}, nil
	}

	// Caso contrário, não tem registros vinculados: deleta definitivamente
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return &ports.DeleteUserResult{
		Action:  ports.UserActionDeleted,
		Message: fmt.Sprintf("Usuário '%s' não possuía compras vinculadas e foi deletado permanentemente com sucesso.", user.Name),
		UserID:  id,
	}, nil
}
