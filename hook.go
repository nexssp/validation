package validation

import (
	"context"
	"reflect"
	"sync"

	"github.com/nexssp/kernel/action"
)

// Hook returns a stateless validation hook suitable for action.Library.Hooks.
// The OnBuild predicate drops the hook from every action whose Req type
// carries no `validate:` tags and does not implement Validatable — such
// actions pay zero cost in Do().
//
// Usage:
//
//	action.MustNewRegistry(action.Library{
//	    Name:    "myapp",
//	    Actions: myActions,
//	    Hooks:   []action.AnyHook{validation.Hook()},
//	})
func Hook() action.AnyHook {
	return action.AnyHook{
		OnBuild: func(_ *action.Meta, reqType, _ reflect.Type) bool {
			return typeNeedsValidation(reqType)
		},
		Before: func(ctx context.Context, req any, _ *action.Meta) (context.Context, error) {
			if req == nil {
				return ctx, nil
			}
			if err := Struct(ctx, req); err != nil {
				return ctx, FromValidatorError(err)
			}
			return ctx, nil
		},
	}
}

var needsValidationCache sync.Map // reflect.Type -> bool

func typeNeedsValidation(reqType reflect.Type) bool {
	for reqType != nil && reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}
	if reqType == nil || reqType.Kind() != reflect.Struct {
		return false
	}
	if cached, ok := needsValidationCache.Load(reqType); ok {
		if needs, ok := cached.(bool); ok {
			return needs
		}
		return false
	}
	needs := implementsValidatable(reqType) || scanForValidateTags(reqType, maxScanDepth)
	needsValidationCache.Store(reqType, needs)
	return needs
}

const maxScanDepth = 32

func implementsValidatable(reqType reflect.Type) bool {
	validatableType := reflect.TypeFor[Validatable]()
	if reqType.Implements(validatableType) {
		return true
	}
	return reflect.PointerTo(reqType).Implements(validatableType)
}

func scanForValidateTags(reqType reflect.Type, remaining int) bool {
	if remaining <= 0 || reqType == nil || reqType.Kind() != reflect.Struct {
		return false
	}
	for field := range reqType.Fields() {
		if field.Tag.Get("validate") != "" {
			return true
		}
		fieldType := field.Type
		for fieldType.Kind() == reflect.Pointer ||
			fieldType.Kind() == reflect.Slice ||
			fieldType.Kind() == reflect.Array {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct && scanForValidateTags(fieldType, remaining-1) {
			return true
		}
	}
	return false
}
