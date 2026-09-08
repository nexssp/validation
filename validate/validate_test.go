package validate_test

import (
	"errors"
	"testing"

	"github.com/nexssp/kernel/xerr"
	"github.com/nexssp/validation/validate"
)

func TestValidateAll(t *testing.T) {
	t.Parallel()

	t.Run("ZeroAllocationPass", func(t *testing.T) {
		err := validate.All(
			validate.Required("production", "env"),
			validate.Min(10, 1, "replica_count"),
			validate.MaxLength("nexss", 20, "name"),
		)
		if err != nil {
			t.Fatalf("expected nil for valid rules, got: %v", err)
		}
	})

	t.Run("CoalescedValidationErrors", func(t *testing.T) {
		err := validate.All(
			validate.Required("", "username"),
			validate.Min(0, 10, "score"),
			validate.MaxLength("too_long_string", 5, "code"),
		)

		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		var appErr *xerr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected *xerr.AppError, got %T", err)
		}

		if len(appErr.ValidationDetails) != 3 {
			t.Fatalf("expected 3 validation details, got %d", len(appErr.ValidationDetails))
		}

		if appErr.ValidationDetails[0].Field != "username" {
			t.Errorf("expected field 'username', got %q", appErr.ValidationDetails[0].Field)
		}
	})
}
