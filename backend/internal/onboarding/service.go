package onboarding

import "context"

type Repository interface {
	Get(context.Context, Identity, string, string) (State, error)
	Update(context.Context, Identity, string, Update, string, string) (State, error)
}

type Service struct {
	repository     Repository
	termsVersion   string
	privacyVersion string
}

func NewService(repository Repository, termsVersion, privacyVersion string) *Service {
	return &Service{repository: repository, termsVersion: termsVersion, privacyVersion: privacyVersion}
}

func (service *Service) Get(ctx context.Context, identity Identity) (State, error) {
	return service.repository.Get(ctx, identity, service.termsVersion, service.privacyVersion)
}

func (service *Service) Update(ctx context.Context, identity Identity, idempotencyKey string, update Update) (State, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return State{}, err
	}
	if err := update.Normalize(service.termsVersion, service.privacyVersion); err != nil {
		return State{}, err
	}
	return service.repository.Update(ctx, identity, idempotencyKey, update, service.termsVersion, service.privacyVersion)
}
