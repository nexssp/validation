package validate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nexssp/kernel/xerr"
)

// All combines multiple zero-reflection validation checks into an xerr error.
func All(checks ...error) error {
	var details xerr.ValidationDetails
	for _, err := range checks {
		if err != nil {
			if ve, ok := errors.AsType[fieldError](err); ok {
				details = append(details, xerr.ValidationDetail{
					Field:      ve.field,
					Validation: ve.rule,
					Value:      ve.msg,
				})
			} else {
				details = append(details, xerr.ValidationDetail{
					Field:      "unknown",
					Validation: "custom",
					Value:      err.Error(),
				})
			}
		}
	}

	if len(details) > 0 {
		return &xerr.AppError{
			Kind:              xerr.KindValidation,
			Message:           "validation failed",
			ValidationDetails: details,
		}
	}
	return nil
}

type fieldError struct {
	field, rule, msg string
}

func (e fieldError) Error() string { return e.msg }

func Required(val, field string) error {
	if strings.TrimSpace(val) == "" {
		return fieldError{field, "required", "field cannot be empty"}
	}
	return nil
}

func Min[T ~int | ~int64 | ~float64](val, minVal T, field string) error {
	if val < minVal {
		return fieldError{field, "min", fmt.Sprintf("must be >= %v", minVal)}
	}
	return nil
}

func MaxLength(val string, maxLen int, field string) error {
	if len(val) > maxLen {
		return fieldError{field, "max_length", fmt.Sprintf("must be <= %d characters", maxLen)}
	}
	return nil
}
