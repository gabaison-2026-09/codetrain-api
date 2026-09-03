package service

import (
	"context"
	"errors"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

// SRS は復習期限の問題一覧を返すユースケース。
type SRS struct {
	userResolver
	repo SRSRepository
}

func NewSRS(users UserRepository, repo SRSRepository) *SRS {
	return &SRS{userResolver: userResolver{repo: users}, repo: repo}
}

func (s *SRS) ListDue(ctx context.Context, externalID string, limit int) ([]domain.SRSDueItem, error) {
	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListDue(ctx, userID, limit)
	if errors.Is(err, repository.ErrNotFound) {
		return []domain.SRSDueItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.SRSDueItem{}
	}
	return items, nil
}
