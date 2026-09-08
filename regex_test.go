package validation_test

import (
	"testing"
	"time"

	"github.com/nexssp/validation"
)

func TestRegexValidators(t *testing.T) {
	t.Parallel()

	t.Run("IsValidEmail", func(t *testing.T) {
		valid := []string{"dev@nexss.com", "user.name+tag@sub.domain.co"}
		invalid := []string{"plainaddress", "@domain.com", "user@.com"}

		for _, email := range valid {
			if !validation.IsValidEmail(email) {
				t.Errorf("expected email %q to be valid", email)
			}
		}
		for _, email := range invalid {
			if validation.IsValidEmail(email) {
				t.Errorf("expected email %q to be invalid", email)
			}
		}
	})

	t.Run("IsValidUUID", func(t *testing.T) {
		if !validation.IsValidUUID("f47ac10b-58cc-4372-a567-0e02b2c3d479") {
			t.Error("expected valid UUID v4")
		}
		if validation.IsValidUUID("not-a-uuid") {
			t.Error("expected invalid UUID")
		}
	})

	t.Run("CompileWithTimeout", func(t *testing.T) {
		re := validation.CompileWithTimeout(`^[a-zA-Z0-9]+$`, 100*time.Millisecond)
		if re == nil {
			t.Fatal("expected regex compilation success")
		}
		if !re.MatchString("Nexss2026") {
			t.Error("expected match")
		}
	})
}
