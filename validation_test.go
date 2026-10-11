package validation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/nexssp/kernel/xerr"

	"github.com/nexssp/validation"
)

type userDTO struct {
	Name  string `json:"name"     validate:"required,min=3"`
	Email string `json:"email"    validate:"required,email"`
}

func TestStructValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ValidStruct", func(t *testing.T) {
		u := userDTO{Name: "Alice", Email: "alice@nexss.com"}
		if err := validation.Struct(ctx, u); err != nil {
			t.Fatalf("expected valid struct, got err: %v", err)
		}
	})

	t.Run("InvalidStructTags", func(t *testing.T) {
		u := userDTO{Name: "Al", Email: "invalid-email"}
		err := validation.Struct(ctx, u)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		var fieldErrors validator.ValidationErrors
		if !errors.As(err, &fieldErrors) {
			t.Fatalf("expected validator.ValidationErrors, got %T", err)
		}
		if len(fieldErrors) != 2 {
			t.Errorf("expected 2 field errors, got %d", len(fieldErrors))
		}
	})

	t.Run("NilAndPointers", func(t *testing.T) {
		if err := validation.Struct(ctx, nil); err != nil {
			t.Errorf("expected nil for nil input, got: %v", err)
		}

		var ptr *userDTO
		if err := validation.Struct(ctx, ptr); err != nil {
			t.Errorf("expected nil for nil pointer, got: %v", err)
		}
	})
}

func TestTrimStringFields_Recursive(t *testing.T) {
	t.Parallel()

	type Address struct {
		Street string
		City   string
	}

	type Person struct {
		Name      string
		Addresses []Address
		Primary   *Address
		age       int // unexported
	}

	p := &Person{
		Name: "  Alice  \t",
		Addresses: []Address{
			{Street: "  123 Main St  ", City: "  Metropolis  "},
		},
		Primary: &Address{
			Street: "  456 Oak Rd  ",
			City:   "  Gotham  ",
		},
		age: 30,
	}

	validation.TrimStringFields(p)

	if p.Name != "Alice" {
		t.Errorf("expected 'Alice', got %q", p.Name)
	}
	if p.Addresses[0].Street != "123 Main St" || p.Addresses[0].City != "Metropolis" {
		t.Errorf("expected trimmed slice address, got %+v", p.Addresses[0])
	}
	if p.Primary.Street != "456 Oak Rd" || p.Primary.City != "Gotham" {
		t.Errorf("expected trimmed pointer address, got %+v", p.Primary)
	}
}

func TestRegisterCustomValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	validation.RegisterCustom("must_be_nexss", func(_ context.Context, fl validator.FieldLevel) bool {
		return fl.Field().String() == "nexss"
	})

	type customTarget struct {
		Code string `json:"code" validate:"must_be_nexss"`
	}

	if err := validation.Struct(ctx, customTarget{Code: "nexss"}); err != nil {
		t.Errorf("expected custom rule pass, got: %v", err)
	}

	if err := validation.Struct(ctx, customTarget{Code: "other"}); err == nil {
		t.Error("expected custom rule failure, got nil")
	}
}

func TestVar_AcceptsValidScalar(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		value any
		rule  string
	}{
		{"required non-empty string", "alice", "required"},
		{"email", "alice@nexss.com", "required,email"},
		{"oneof hit", "medium", "oneof=low medium high"},
		{"int gt", 5, "gt=0"},
		{"empty rule is noop", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := validation.Var(ctx, tc.value, tc.rule); err != nil {
				t.Fatalf("Var(%v, %q) = %v, want nil", tc.value, tc.rule, err)
			}
		})
	}
}

func TestVar_RejectsInvalidScalar(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		value   any
		rule    string
		wantTag string
	}{
		{"empty required", "", "required", "required"},
		{"bad email", "not-an-email", "required,email", "email"},
		{"oneof miss", "urgent", "oneof=low medium high", "oneof"},
		{"int gte", -1, "gte=0", "gte"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validation.Var(ctx, tc.value, tc.rule)
			if err == nil {
				t.Fatalf("Var(%v, %q) = nil, want error", tc.value, tc.rule)
			}
			if kind := xerr.KindFrom(err); kind != xerr.KindValidation {
				t.Fatalf("KindFrom = %v, want KindValidation", kind)
			}
			appErr, ok := errors.AsType[*xerr.AppError](err)
			if !ok || len(appErr.ValidationDetails) == 0 {
				t.Fatalf("missing structured detail: %+v", err)
			}
			if got := appErr.ValidationDetails[0].Validation; got != tc.wantTag {
				t.Errorf("Validation = %q, want %q", got, tc.wantTag)
			}
		})
	}
}

func TestMap_CollectsPerKeyDetails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	err := validation.Map(ctx, map[string]any{
		"email":   "not-an-email",
		"urgency": "urgent",
		"count":   5,
	}, map[string]any{
		"email":   "required,email",
		"urgency": "required,oneof=low medium high critical",
		"count":   "gt=0",
	})
	if err == nil {
		t.Fatal("Map returned nil, want error")
	}

	appErr, ok := errors.AsType[*xerr.AppError](err)
	if !ok {
		t.Fatalf("err type = %T, want *xerr.AppError", err)
	}
	if len(appErr.ValidationDetails) != 2 {
		t.Fatalf("details = %d, want 2", len(appErr.ValidationDetails))
	}

	byField := map[string]string{}
	for _, d := range appErr.ValidationDetails {
		byField[d.Field] = d.Validation
	}
	if byField["email"] != "email" {
		t.Errorf("email tag = %q, want email", byField["email"])
	}
	if byField["urgency"] != "oneof" {
		t.Errorf("urgency tag = %q, want oneof", byField["urgency"])
	}
}

func TestMap_NilAndEmptyRulesAreNoop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	if err := validation.Map(ctx, nil, nil); err != nil {
		t.Errorf("Map(nil, nil) = %v, want nil", err)
	}
	if err := validation.Map(ctx, map[string]any{"x": "y"}, map[string]any{}); err != nil {
		t.Errorf("Map(_, empty) = %v, want nil", err)
	}
}
