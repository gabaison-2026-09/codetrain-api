package service

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// MeStatsService は GET /v1/me/stats のユースケース。
// 認証ユーザーの種別×言語別 累計回答数・正答率を返す。
type MeStatsService struct {
	userResolver
	repo MeStatsRepository
}

func NewMeStats(users UserRepository, repo MeStatsRepository) *MeStatsService {
	return &MeStatsService{userResolver: userResolver{repo: users}, repo: repo}
}

// Stats は sub からユーザーを解決し、user_type_stat の各行に accuracy を付けて返す。
// 未回答ユーザーは空スライスを返す。
func (s *MeStatsService) Stats(ctx context.Context, externalID string) ([]domain.TypeStat, error) {
	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return nil, err
	}
	stats, err := s.repo.ListTypeStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range stats {
		if stats[i].Attempts > 0 {
			stats[i].Accuracy = float64(stats[i].Corrects) / float64(stats[i].Attempts)
		}
	}
	if stats == nil {
		stats = []domain.TypeStat{}
	}
	return stats, nil
}
