package users

import "context"

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Get(ctx context.Context, identity Identity) (Profile, error) {
	return s.repository.Get(ctx, identity)
}

func (s *Service) Update(ctx context.Context, identity Identity, update Update) (Profile, error) {
	displayName, err := NormalizeDisplayName(update.DisplayName)
	if err != nil {
		return Profile{}, err
	}
	if update.ExpectedVersion < 1 {
		return Profile{}, ErrConflict
	}
	update.DisplayName = displayName
	return s.repository.Update(ctx, identity, update)
}
