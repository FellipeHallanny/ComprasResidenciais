package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type analyticsService struct {
	purchaseRepo ports.PurchaseRepository
	productRepo  ports.CanonicalProductRepository
}

// NewAnalyticsService cria instância do serviço de analytics
func NewAnalyticsService(
	purchaseRepo ports.PurchaseRepository,
	productRepo ports.CanonicalProductRepository,
) ports.AnalyticsService {
	return &analyticsService{
		purchaseRepo: purchaseRepo,
		productRepo:  productRepo,
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
	return s.productRepo.SearchByName(ctx, query)
}

// --- User Service ---

type userService struct {
	userRepo ports.UserRepository
}

// NewUserService cria instância do serviço de usuários
func NewUserService(userRepo ports.UserRepository) ports.UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) CreateUser(ctx context.Context, name, email string) (*domain.User, error) {
	user := &domain.User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		IsBlocked: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.userRepo.ListAll(ctx)
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
