package validation

import (
	"context"
	"errors"
	"reflect"
	"strings"

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

// FromValidatorError maps validator/v10 field errors to Nexss
// structured details with short, human-readable messages.
//
// Each ValidationDetail carries:
//
//   - Field:      the JSON name of the failing field
//   - Validation: the raw validator tag (e.g. "oneof", "min")
//   - Value:      a short phrase suitable for a per-field error line,
//     written the way a user would describe the problem
//
// The Validation tag is preserved verbatim so callers that branch on
// it (metrics, i18n, dashboards) keep working unchanged. Only Value
// is humanized.
func FromValidatorError(err error) *xerr.AppError {
	appErr := xerr.Validation("validation failed", err)
	if fieldErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
		details := make(xerr.ValidationDetails, 0, len(fieldErrors))
		for _, fieldErr := range fieldErrors {
			details = append(details, xerr.ValidationDetail{
				Field:      fieldErr.Field(),
				Validation: fieldErr.Tag(),
				Value:      humanizeValidatorTag(fieldErr),
			})
		}
		appErr.ValidationDetails = details
	}
	return appErr
}

// validatorStaticMessages maps validator/v10 tags that have a fixed
// human-readable phrase to that phrase. Tags whose message depends on
// the FieldError's param or Kind are handled by humanizeParametricTag.
var validatorStaticMessages = map[string]string{
	"required": "is required",

	"email": "must be a valid email address",
	"url":   "must be a valid URL",
	"uri":   "must be a valid URI",
	"uuid":  "must be a valid UUID",
	"uuid4": "must be a valid UUID",

	"alphanum": "must contain only letters and digits",
	"alpha":    "must contain only letters",
	"numeric":  "must contain only digits",
}

// humanizeValidatorTag converts a validator/v10 FieldError into a
// short, user-facing phrase. Static tags are answered by a map lookup;
// parametric and length-aware tags go through humanizeParametricTag.
// Unknown tags fall back to the raw "failed '<tag>' check" form so
// nothing is silently swallowed.
func humanizeValidatorTag(fe validator.FieldError) string {
	tag := fe.Tag()
	if msg, ok := validatorStaticMessages[tag]; ok {
		return msg
	}
	return humanizeParametricTag(fe, tag)
}

// humanizeParametricTag handles every tag whose message depends on
// fe.Param() or fe.Kind().
func humanizeParametricTag(fe validator.FieldError, tag string) string {
	param := fe.Param()

	switch tag {
	case "oneof":
		return humanizeOneOf(param)

	case "min", "gte":
		return humanizeBound("at least", param, isLengthKind(fe.Kind()))
	case "max", "lte":
		return humanizeBound("at most", param, isLengthKind(fe.Kind()))
	case "len":
		return humanizeBound("exactly", param, isLengthKind(fe.Kind()))

	case "gt":
		return "must be greater than " + param
	case "lt":
		return "must be less than " + param
	case "eq":
		return "must equal " + param
	case "ne":
		return "must not equal " + param

	default:
		return fallbackValidatorMessage(tag, param)
	}
}

// humanizeOneOf renders the "oneof=a b c" tag as a comma-separated
// list, or as a generic phrase when the tag carries no param.
func humanizeOneOf(param string) string {
	if param == "" {
		return "must be one of the allowed values"
	}
	return "must be one of: " + strings.ReplaceAll(param, " ", ", ")
}

// humanizeBound renders the min/max/len family. length selects the
// "N characters" suffix for string/slice/array kinds; numeric kinds
// get the bare bound.
func humanizeBound(bound, param string, length bool) string {
	if length {
		return "must be " + bound + " " + param + " characters"
	}
	return "must be " + bound + " " + param
}

// fallbackValidatorMessage renders a tag the humanizer does not know,
// preserving the tag and param so nothing is lost.
func fallbackValidatorMessage(tag, param string) string {
	if param != "" {
		return "failed '" + tag + "' check (" + param + ")"
	}
	return "failed '" + tag + "' check"
}

// isLengthKind reports whether a validator Kind refers to a
// length-bearing value, so "min=3" can render as "at least 3
// characters" instead of "at least 3".
func isLengthKind(k reflect.Kind) bool {
	return k == reflect.String ||
		k == reflect.Slice ||
		k == reflect.Array ||
		k == reflect.Map ||
		k == reflect.Chan
}
