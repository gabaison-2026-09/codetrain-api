package service

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

type CalendarService struct {
	userResolver
	repo CalendarRepository
}

func NewCalendar(users UserRepository, repo CalendarRepository) *CalendarService {
	return &CalendarService{userResolver: userResolver{repo: users}, repo: repo}
}

func (s *CalendarService) Get(ctx context.Context, externalID, from, to string) (domain.Calendar, error) {
	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return domain.Calendar{}, err
	}
	days, err := s.repo.DailyConsumption(ctx, userID, from, to)
	if err != nil {
		return domain.Calendar{}, err
	}
	streak, last, err := s.repo.Streak(ctx, userID)
	if err != nil {
		return domain.Calendar{}, err
	}
	if days == nil {
		days = []domain.CalendarDay{}
	}
	return domain.Calendar{Days: days, StreakDays: streak, LastStudiedOn: last}, nil
}
