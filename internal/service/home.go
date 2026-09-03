package service

import (
	"context"
	"errors"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

var ErrNoAvailableQuestion = errors.New("利用可能な問題がありません")

type HomeService struct {
	userResolver
	repo HomeRepository
}

func NewHome(users UserRepository, repo HomeRepository) *HomeService {
	return &HomeService{userResolver: userResolver{repo: users}, repo: repo}
}

func (s *HomeService) Get(ctx context.Context, externalID string) (domain.Home, error) {
	user, progress, err := s.repoUser(ctx, externalID)
	if err != nil {
		return domain.Home{}, err
	}
	slots, err := s.repo.ListUserTasks(ctx, user.ID)
	if err != nil {
		return domain.Home{}, err
	}
	for _, slot := range slots {
		difficulty := 3
		if slot.Difficulty != nil {
			difficulty = *slot.Difficulty
		}
		slot.Difficulty = &difficulty
		q, err := s.repo.PickUnansweredPublished(ctx, user.ID, slot.QuestionType, slot.Language, difficulty)
		if errors.Is(err, repository.ErrNotFound) {
			continue
		}
		if err != nil {
			return domain.Home{}, err
		}
		if err := s.repo.UpsertDailyTask(ctx, user.ID, slot, q.ID); err != nil {
			return domain.Home{}, err
		}
	}
	tasks, err := s.repo.GetTodayHome(ctx, user.ID)
	if err != nil {
		return domain.Home{}, err
	}
	if tasks == nil {
		tasks = []domain.HomeTask{}
	}
	return domain.Home{Tasks: tasks, Progress: progress}, nil
}

func (s *HomeService) repoUser(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	return s.resolveUser(ctx, externalID)
}
