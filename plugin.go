package validation

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xerr"
)

// Validatable allows structs to execute custom semantic validation logic.
type Validatable interface {
	Validate() error
}

// New creates an action plugin using Copy-on-Write cache reads on the hot path.
func New() action.AnyAction {
	var cache atomic.Pointer[map[string]bool]
	cache.Store(&map[string]bool{})
	var mu sync.Mutex

	return action.New[any, any]("plugin.validator", nil).
		AnyHook(action.AnyHook{
			Before: func(ctx context.Context, req any, meta *action.Meta) (context.Context, error) {
				if req == nil {
					return ctx, nil
				}

				// 1. HOT PATH: Lock-free atomic map lookup
				needsMap := *cache.Load()
				needs, known := needsMap[meta.Name]

				if !known {
					// 2. COLD PATH: Inspect type once
					mu.Lock()
					needsMap = *cache.Load()
					needs, known = needsMap[meta.Name]
					if !known {
						needs = inferNeedsValidation(req)
						updated := make(map[string]bool, len(needsMap)+1)
						for k, v := range needsMap {
							updated[k] = v
						}
						updated[meta.Name] = needs
						cache.Store(&updated)
					}
					mu.Unlock()
				}

				if !needs {
					return ctx, nil
				}

				// 3. EXECUTE VALIDATION
				TrimStringFields(req)

				if v, ok := req.(Validatable); ok {
					if err := v.Validate(); err != nil {
						var appErr *xerr.AppError
						if errors.As(err, &appErr) {
							return ctx, appErr
						}
						return ctx, xerr.Validation("custom validation failed", err)
					}
				}

				if err := Struct(ctx, req); err != nil {
					return ctx, FromValidatorError(err)
				}

				return ctx, nil
			},
		}).
		Build()
}

func inferNeedsValidation(req any) bool {
	if _, ok := req.(Validatable); ok {
		return true
	}
	t := reflect.TypeOf(req)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct && hasValidateTags(t)
}

func hasValidateTags(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("validate") != "" {
			return true
		}
		ft := field.Type
		for ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && hasValidateTags(ft) {
			return true
		}
	}
	return false
}
