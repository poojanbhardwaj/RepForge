package onboarding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound            = errors.New("onboarding not found")
	ErrValidation          = errors.New("onboarding validation failed")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{16,128}$`)
var timezonePattern = regexp.MustCompile(`^(UTC|[A-Za-z_]+(?:/[A-Za-z0-9_+\-]+)+)$`)

var allowedEquipment = map[string]struct{}{
	"bodyweight": {}, "dumbbells": {}, "barbell": {}, "rack": {},
	"bench": {}, "cables": {}, "machines": {}, "bands": {},
}

type Identity struct {
	Provider string
	Subject  string
}

type Update struct {
	Source                 string   `json:"-"`
	AdultAttested          *bool    `json:"adultAttested,omitempty"`
	TermsVersion           *string  `json:"termsVersion,omitempty"`
	PrivacyVersion         *string  `json:"privacyVersion,omitempty"`
	Timezone               *string  `json:"timezone,omitempty"`
	Units                  *string  `json:"units,omitempty"`
	PrimaryGoal            *string  `json:"primaryGoal,omitempty"`
	ExperienceLevel        *string  `json:"experienceLevel,omitempty"`
	WeeklyAvailability     *int16   `json:"weeklyAvailability,omitempty"`
	SessionDurationMinutes *int16   `json:"sessionDurationMinutes,omitempty"`
	EquipmentAccess        []string `json:"equipmentAccess,omitempty"`
	DietPreference         *string  `json:"dietPreference,omitempty"`
	ClearDietPreference    bool     `json:"-"`
	SafetyAcknowledged     *bool    `json:"safetyAcknowledged,omitempty"`
	CurrentStep            *int16   `json:"currentStep,omitempty"`
}

var updateJSONFields = map[string]struct{}{
	"adultAttested": {}, "termsVersion": {}, "privacyVersion": {}, "timezone": {},
	"units": {}, "primaryGoal": {}, "experienceLevel": {}, "weeklyAvailability": {},
	"sessionDurationMinutes": {}, "equipmentAccess": {}, "dietPreference": {},
	"safetyAcknowledged": {}, "currentStep": {},
}

func (update *Update) UnmarshalJSON(data []byte) error {
	type wireUpdate Update
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for field := range fields {
		if _, allowed := updateJSONFields[field]; !allowed {
			return fmt.Errorf("unknown onboarding field %q", field)
		}
	}
	var decoded wireUpdate
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*update = Update(decoded)
	if raw, present := fields["dietPreference"]; present && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		update.ClearDietPreference = true
	}
	return nil
}

func (update Update) MarshalJSON() ([]byte, error) {
	type wireUpdate Update
	encoded, err := json.Marshal(wireUpdate(update))
	if err != nil || !update.ClearDietPreference {
		return encoded, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	fields["dietPreference"] = json.RawMessage("null")
	return json.Marshal(fields)
}

type State struct {
	UserID                 uuid.UUID  `json:"userId"`
	State                  string     `json:"state"`
	CurrentStep            int16      `json:"currentStep"`
	Version                int64      `json:"version"`
	AdultAttestedAt        *time.Time `json:"adultAttestedAt"`
	TermsVersion           *string    `json:"termsVersion"`
	TermsAcceptedAt        *time.Time `json:"termsAcceptedAt"`
	PrivacyVersion         *string    `json:"privacyVersion"`
	PrivacyAcceptedAt      *time.Time `json:"privacyAcceptedAt"`
	Timezone               *string    `json:"timezone"`
	Units                  *string    `json:"units"`
	PrimaryGoal            *string    `json:"primaryGoal"`
	ExperienceLevel        *string    `json:"experienceLevel"`
	WeeklyAvailability     *int16     `json:"weeklyAvailability"`
	SessionDurationMinutes *int16     `json:"sessionDurationMinutes"`
	EquipmentAccess        []string   `json:"equipmentAccess"`
	DietPreference         *string    `json:"dietPreference"`
	SafetyAcknowledgedAt   *time.Time `json:"safetyAcknowledgedAt"`
	CompletedAt            *time.Time `json:"completedAt"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
	RequiredTermsVersion   string     `json:"requiredTermsVersion"`
	RequiredPrivacyVersion string     `json:"requiredPrivacyVersion"`
}

func ValidateIdempotencyKey(value string) error {
	if !idempotencyKeyPattern.MatchString(value) {
		return fmt.Errorf("%w: Idempotency-Key must be 16 to 128 safe ASCII characters", ErrValidation)
	}
	return nil
}

func (update *Update) Normalize(termsVersion, privacyVersion string) error {
	if update.Source != "mobile" && update.Source != "web" && update.Source != "synthetic_local" {
		return fmt.Errorf("%w: source is invalid", ErrValidation)
	}
	if update.AdultAttested != nil && !*update.AdultAttested {
		return fmt.Errorf("%w: adult attestation must be accepted to continue", ErrValidation)
	}
	if update.SafetyAcknowledged != nil && !*update.SafetyAcknowledged {
		return fmt.Errorf("%w: safety acknowledgement must be accepted to continue", ErrValidation)
	}
	if update.TermsVersion != nil && *update.TermsVersion != termsVersion {
		return fmt.Errorf("%w: terms version is no longer current", ErrValidation)
	}
	if update.PrivacyVersion != nil && *update.PrivacyVersion != privacyVersion {
		return fmt.Errorf("%w: privacy version is no longer current", ErrValidation)
	}
	if update.Timezone != nil {
		value := strings.TrimSpace(*update.Timezone)
		if len(value) > 64 || !timezonePattern.MatchString(value) {
			return fmt.Errorf("%w: timezone must be a valid IANA timezone", ErrValidation)
		}
		if _, err := time.LoadLocation(value); err != nil {
			return fmt.Errorf("%w: timezone must be a valid IANA timezone", ErrValidation)
		}
		update.Timezone = &value
	}
	if update.Units != nil && *update.Units != "metric" && *update.Units != "imperial" {
		return fmt.Errorf("%w: units are invalid", ErrValidation)
	}
	if update.PrimaryGoal != nil && *update.PrimaryGoal != "strength" && *update.PrimaryGoal != "muscle" && *update.PrimaryGoal != "general_fitness" {
		return fmt.Errorf("%w: primary goal is invalid", ErrValidation)
	}
	if update.ExperienceLevel != nil && *update.ExperienceLevel != "beginner" && *update.ExperienceLevel != "intermediate" {
		return fmt.Errorf("%w: experience level is invalid", ErrValidation)
	}
	if update.WeeklyAvailability != nil && (*update.WeeklyAvailability < 1 || *update.WeeklyAvailability > 7) {
		return fmt.Errorf("%w: weekly availability must be between 1 and 7", ErrValidation)
	}
	if update.SessionDurationMinutes != nil && (*update.SessionDurationMinutes < 15 || *update.SessionDurationMinutes > 180) {
		return fmt.Errorf("%w: session duration must be between 15 and 180 minutes", ErrValidation)
	}
	if update.EquipmentAccess != nil {
		if len(update.EquipmentAccess) < 1 || len(update.EquipmentAccess) > 8 {
			return fmt.Errorf("%w: choose between 1 and 8 equipment options", ErrValidation)
		}
		seen := make(map[string]struct{}, len(update.EquipmentAccess))
		for _, item := range update.EquipmentAccess {
			if _, ok := allowedEquipment[item]; !ok {
				return fmt.Errorf("%w: equipment access is invalid", ErrValidation)
			}
			if _, duplicate := seen[item]; duplicate {
				return fmt.Errorf("%w: equipment access must not contain duplicates", ErrValidation)
			}
			seen[item] = struct{}{}
		}
		update.EquipmentAccess = update.EquipmentAccess[:0]
		for item := range seen {
			update.EquipmentAccess = append(update.EquipmentAccess, item)
		}
		sort.Strings(update.EquipmentAccess)
	}
	if update.ClearDietPreference && update.DietPreference != nil {
		return fmt.Errorf("%w: diet preference cannot be set and cleared together", ErrValidation)
	}
	if update.DietPreference != nil && *update.DietPreference != "vegetarian" && *update.DietPreference != "eggetarian" && *update.DietPreference != "vegan" && *update.DietPreference != "omnivore" {
		return fmt.Errorf("%w: diet preference is invalid", ErrValidation)
	}
	if update.CurrentStep != nil && (*update.CurrentStep < 1 || *update.CurrentStep > 6) {
		return fmt.Errorf("%w: current step must be between 1 and 6", ErrValidation)
	}
	if !update.hasProgress() {
		return fmt.Errorf("%w: at least one onboarding field is required", ErrValidation)
	}
	return nil
}

func (update Update) hasProgress() bool {
	return update.AdultAttested != nil || update.TermsVersion != nil || update.PrivacyVersion != nil ||
		update.Timezone != nil || update.Units != nil || update.PrimaryGoal != nil ||
		update.ExperienceLevel != nil || update.WeeklyAvailability != nil ||
		update.SessionDurationMinutes != nil || update.EquipmentAccess != nil ||
		update.DietPreference != nil || update.ClearDietPreference || update.SafetyAcknowledged != nil || update.CurrentStep != nil
}
