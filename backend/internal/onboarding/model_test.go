package onboarding

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestUpdateNormalizeCanonicalizesEquipmentAndAcceptsCompleteInput(t *testing.T) {
	adult, safety := true, true
	terms, privacy := "terms-1", "privacy-1"
	timezone, units := " Asia/Kolkata ", "metric"
	goal, experience, diet := "strength", "beginner", "vegetarian"
	weekly, duration, step := int16(4), int16(60), int16(6)
	update := Update{
		Source: "web", AdultAttested: &adult, TermsVersion: &terms, PrivacyVersion: &privacy,
		Timezone: &timezone, Units: &units, PrimaryGoal: &goal, ExperienceLevel: &experience,
		WeeklyAvailability: &weekly, SessionDurationMinutes: &duration,
		EquipmentAccess: []string{"dumbbells", "bodyweight"}, DietPreference: &diet,
		SafetyAcknowledged: &safety, CurrentStep: &step,
	}
	if err := update.Normalize("terms-1", "privacy-1"); err != nil {
		t.Fatal(err)
	}
	if *update.Timezone != "Asia/Kolkata" {
		t.Fatalf("timezone = %q", *update.Timezone)
	}
	if !reflect.DeepEqual(update.EquipmentAccess, []string{"bodyweight", "dumbbells"}) {
		t.Fatalf("equipment = %#v", update.EquipmentAccess)
	}
}

func TestUpdateJSONDistinguishesAnOmittedDietFromAnExplicitClear(t *testing.T) {
	var clear Update
	if err := json.Unmarshal([]byte(`{"dietPreference":null}`), &clear); err != nil {
		t.Fatal(err)
	}
	clear.Source = "web"
	if !clear.ClearDietPreference || clear.DietPreference != nil {
		t.Fatalf("clear intent was lost: %+v", clear)
	}
	if err := clear.Normalize("terms-1", "privacy-1"); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(clear)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil || string(fields["dietPreference"]) != "null" {
		t.Fatalf("encoded clear=%s error=%v", encoded, err)
	}

	var omitted Update
	if err := json.Unmarshal([]byte(`{"currentStep":2}`), &omitted); err != nil {
		t.Fatal(err)
	}
	if omitted.ClearDietPreference {
		t.Fatal("omitted diet was interpreted as a clear")
	}
	if err := json.Unmarshal([]byte(`{"diagnosis":"synthetic"}`), &omitted); err == nil {
		t.Fatal("expected unknown field rejection")
	}
}

func TestUpdateNormalizeRejectsSensitiveOrInvalidShapes(t *testing.T) {
	falseValue := false
	badTerms, badTimezone, badUnits := "old-terms", "Not/A_Timezone", "stones"
	zero, tooLong, badStep := int16(0), int16(181), int16(7)
	tests := map[string]Update{
		"empty":                    {Source: "web"},
		"unknown source":           {Source: "browser", AdultAttested: boolPointer(true)},
		"negative attestation":     {Source: "web", AdultAttested: &falseValue},
		"stale terms":              {Source: "web", TermsVersion: &badTerms},
		"invalid timezone":         {Source: "web", Timezone: &badTimezone},
		"invalid units":            {Source: "web", Units: &badUnits},
		"invalid availability":     {Source: "web", WeeklyAvailability: &zero},
		"invalid session duration": {Source: "web", SessionDurationMinutes: &tooLong},
		"invalid equipment":        {Source: "web", EquipmentAccess: []string{"medical_diagnosis"}},
		"duplicate equipment":      {Source: "web", EquipmentAccess: []string{"bodyweight", "bodyweight"}},
		"invalid current step":     {Source: "web", CurrentStep: &badStep},
	}
	for name, update := range tests {
		t.Run(name, func(t *testing.T) {
			if err := update.Normalize("terms-1", "privacy-1"); !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want validation", err)
			}
		})
	}
}

func TestValidateIdempotencyKey(t *testing.T) {
	for _, key := range []string{"web-123456789012", "12345678-1234-4234-8234-123456789012"} {
		if err := ValidateIdempotencyKey(key); err != nil {
			t.Fatalf("valid key %q: %v", key, err)
		}
	}
	for _, key := range []string{"short", "unsafe key has spaces", "line-break-123456\n"} {
		if err := ValidateIdempotencyKey(key); !errors.Is(err, ErrValidation) {
			t.Fatalf("invalid key %q error = %v", key, err)
		}
	}
}

func boolPointer(value bool) *bool { return &value }
