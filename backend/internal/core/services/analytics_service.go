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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"
)

// --- AnalyticsService Implementation ---

type analyticsService struct {
	purchaseRepo ports.PurchaseRepository
	productRepo  ports.CanonicalProductRepository
	embedder     ports.EmbeddingProvider
	embedCache   sync.Map           // cache in-memory de embeddings: map[string]pgvector.Vector
	sfGroup      singleflight.Group // deduplicação de chamadas concorrentes à API de embeddings
}

// NewAnalyticsService cria instância do serviço analítico
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

func (s *analyticsService) GetCatalogWithPrices(ctx context.Context) ([]domain.CatalogItemView, error) {
	return s.purchaseRepo.GetCatalogWithPrices(ctx)
}

// SearchProducts busca produtos canônicos por similaridade semântica ou busca textual.
// OTIMIZAÇÃO: Cache in-memory com singleflight para evitar chamadas de rede redundantes
// ao Gemini em palavras-chave frequentes (reduz latência de 350ms para < 5ms).
func (s *analyticsService) SearchProducts(ctx context.Context, query string) ([]domain.CanonicalProduct, error) {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" {
		return s.productRepo.ListAll(ctx)
	}

	// 1. Tenta busca semântica vetorial via pgvector
	if s.embedder != nil {
		var vec pgvector.Vector
		var err error

		if cached, ok := s.embedCache.Load(trimmed); ok {
			vec = cached.(pgvector.Vector)
		} else {
			res, sfErr, _ := s.sfGroup.Do(trimmed, func() (any, error) {
				v, e := s.embedder.GenerateEmbedding(ctx, trimmed)
				if e == nil {
					s.embedCache.Store(trimmed, v)
				}
				return v, e
			})
			if sfErr == nil {
				vec = res.(pgvector.Vector)
			}
			err = sfErr
		}

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

func (s *userService) CreateUser(ctx context.Context, name, email, phone, password, role string) (*domain.User, error) {
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
		Name:         strings.TrimSpace(name),
		Email:        strings.TrimSpace(email),
		Phone:        strings.TrimSpace(phone),
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

func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, name, email, phone, role string, isBlocked bool, newPassword string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Name = strings.TrimSpace(name)
	user.Email = strings.TrimSpace(email)
	user.Phone = strings.TrimSpace(phone)
	if role != "" {
		user.Role = role
	}
	user.IsBlocked = isBlocked
	if strings.TrimSpace(newPassword) != "" {
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
		return nil, err
	}

	// 2. Verifica se o usuário possui compras vinculadas no histórico
	count, err := s.userRepo.CountPurchasesByUserID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check user purchases: %w", err)
	}

	// 3. Se possui compras vinculadas, NÃO exclui para manter integridade histórica; aplica soft-lock (is_blocked = true)
	if count > 0 {
		if err := s.userRepo.Block(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to block user: %w", err)
		}
		return &ports.DeleteUserResult{
			Action:  ports.UserActionBlocked,
			Message: fmt.Sprintf("O morador '%s' possui %d compra(s) registrada(s). Para manter o histórico fiscal e comparativos, o acesso foi bloqueado em vez de excluído.", user.Name, count),
			UserID:  id,
		}, nil
	}

	// 4. Se não possui compras, exclui fisicamente da tabela
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return &ports.DeleteUserResult{
		Action:  ports.UserActionDeleted,
		Message: fmt.Sprintf("Morador '%s' removido com sucesso.", user.Name),
		UserID:  id,
	}, nil
}
