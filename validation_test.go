package validation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/nexssp/validation"
)

type userDTO struct {
	Name  string `json:"name" validate:"required,min=3"`
	Email string `json:"email" validate:"required,email"`
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
