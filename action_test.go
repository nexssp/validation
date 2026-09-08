package validation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xerr"
	"github.com/nexssp/validation"
)

type orderDTO struct {
	ID    string `json:"id" validate:"required"`
	Count int    `json:"count" validate:"gt=0"`
}

func TestValidateAction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	act := validation.ValidateAction[orderDTO]("test.validate_action")

	// Valid input
	out, err := act.Do(ctx, orderDTO{ID: "ord_123", Count: 5})
	if err != nil {
		t.Fatalf("expected pass, got: %v", err)
	}
	if out.ID != "ord_123" {
		t.Errorf("expected ord_123, got %s", out.ID)
	}

	// Invalid input
	_, err = act.Do(ctx, orderDTO{ID: "", Count: 0})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var appErr *xerr.AppError
	if !errors.As(err, &appErr) || appErr.Kind != xerr.KindValidation {
		t.Fatalf("expected xerr.KindValidation, got %v", err)
	}

	if len(appErr.ValidationDetails) != 2 {
		t.Fatalf("expected 2 validation details, got %d", len(appErr.ValidationDetails))
	}

	if appErr.ValidationDetails[0].Field != "id" || appErr.ValidationDetails[0].Validation != "required" {
		t.Errorf("unexpected detail 0: %+v", appErr.ValidationDetails[0])
	}
	if appErr.ValidationDetails[1].Field != "count" || appErr.ValidationDetails[1].Validation != "gt" {
		t.Errorf("unexpected detail 1: %+v", appErr.ValidationDetails[1])
	}
}

func TestAutoValidate_PopulatesValidationDetails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	act := validation.AutoValidate(
		action.New("order.process", func(_ context.Context, req orderDTO) (string, error) {
			return req.ID, nil
		}),
	).Build()

	// 1. Valid execution
	if _, err := act.Do(ctx, orderDTO{ID: "ord_123", Count: 1}); err != nil {
		t.Fatalf("expected pass, got: %v", err)
	}

	// 2. Invalid execution -> Assert structured details are present
	_, err := act.Do(ctx, orderDTO{ID: "", Count: -1})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var appErr *xerr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *xerr.AppError, got %T", err)
	}

	if len(appErr.ValidationDetails) != 2 {
		t.Fatalf("expected 2 structured validation details, got %d", len(appErr.ValidationDetails))
	}
}
