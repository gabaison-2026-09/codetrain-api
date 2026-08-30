package service

import "context"

// Health は /healthz のユースケース。
type Health struct {
	repo HealthRepository
}

func NewHealth(repo HealthRepository) *Health { return &Health{repo: repo} }

// Check は DB に到達できるかを確かめる。
func (s *Health) Check(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
