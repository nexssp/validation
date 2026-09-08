package validation

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xerr"
)

// ValidateAction creates an identity action that validates T using validate tags.
func ValidateAction[T any](name string) *action.BuiltAction[T, T] {
	return action.New(name, func(_ context.Context, value T) (T, error) {
		return value, nil
	}).Validate(func(ctx context.Context, value T) error {
		TrimStringFields(value)
		if err := Struct(ctx, value); err != nil {
			return FromValidatorError(err)
		}
		return nil
	}).Build()
}

// AutoValidate wires Kernel's .Validate(...) builder contract directly to validation.Struct,
// mapping validator/v10 errors into clean xerr.ValidationDetails.
func AutoValidate[Req, Res any](b *action.Builder[Req, Res]) *action.Builder[Req, Res] {
	return b.Validate(func(ctx context.Context, req Req) error {
		TrimStringFields(req)
		if err := Struct(ctx, req); err != nil {
			return FromValidatorError(err)
		}
		return nil
	})
}

// FromValidatorError maps validator/v10 field errors to Nexss structured details with readable messages.
func FromValidatorError(err error) *xerr.AppError {
	appErr := xerr.Validation("validation failed", err)
	var fieldErrors validator.ValidationErrors
	if errors.As(err, &fieldErrors) {
		details := make(xerr.ValidationDetails, 0, len(fieldErrors))
		for _, fieldErr := range fieldErrors {
			msg := fmt.Sprintf("failed '%s' check", fieldErr.Tag())
			if param := fieldErr.Param(); param != "" {
				msg = fmt.Sprintf("failed '%s' check (expected %s)", fieldErr.Tag(), param)
			}
			details = append(details, xerr.ValidationDetail{
				Field:      fieldErr.Field(),
				Validation: fieldErr.Tag(),
				Value:      msg,
			})
		}
		appErr.ValidationDetails = details
	}
	return appErr
}
