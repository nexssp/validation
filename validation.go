package validation

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	coreMu sync.RWMutex
	core   = sync.OnceValue(func() *validator.Validate {
		v := validator.New(validator.WithRequiredStructEnabled())
		v.RegisterTagNameFunc(func(field reflect.StructField) string {
			name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
		return v
	})
)

// Struct validates exported fields using `validate:"..."` tags.
// Automatically trims whitespace, evaluates tags, and calls Validatable.Validate() if implemented.
func Struct(ctx context.Context, value any) error {
	if value == nil {
		return nil
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	// 1. Sanitization pre-pass: Trim whitespace
	TrimStringFields(value)

	// 2. Struct tag validation
	coreMu.RLock()
	err := core().StructCtx(ctx, value)
	coreMu.RUnlock()
	if err != nil {
		return err
	}

	// 3. Custom semantic validation interface (runs after tag checks pass)
	if v, ok := value.(Validatable); ok {
		return v.Validate()
	}

	return nil
}

// RegisterCustom registers context-aware validation rules safely at cold-path init.
func RegisterCustom(tag string, fn validator.FuncCtx) {
	coreMu.Lock()
	_ = core().RegisterValidationCtx(tag, fn)
	coreMu.Unlock()
}

// RegisterType registers custom type handlers safely at cold-path init.
func RegisterType(fn validator.CustomTypeFunc, types ...any) {
	coreMu.Lock()
	core().RegisterCustomTypeFunc(fn, types...)
	coreMu.Unlock()
}

// TrimStringFields recurses through exported string fields in structs, pointers,
// and slices, trimming leading and trailing whitespace in-place.
func TrimStringFields(value any) {
	if value == nil {
		return
	}
	rv := reflect.ValueOf(value)
	trimValue(rv)
}

func trimValue(rv reflect.Value) {
	if !rv.IsValid() {
		return
	}
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !rv.IsNil() {
			trimValue(rv.Elem())
		}
	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)
			if field.Kind() == reflect.String {
				if field.CanSet() {
					field.SetString(strings.TrimSpace(field.String()))
				}
			} else {
				trimValue(field)
			}
		}
	case reflect.Slice, reflect.Array:
		for j := 0; j < rv.Len(); j++ {
			trimValue(rv.Index(j))
		}
	}
}
