# nexssp/validation

`nexssp/validation` provides high-performance struct and field validation for the Nexss ecosystem using `github.com/go-playground/validator/v10`.

## 🚀 Usage Patterns

### 1. Direct Kernel Action Integration (`AutoValidate`)

```go
validatedAction := validation.AutoValidate(
    action.New("goal.create", handleGoal),
).Build()
```

### 2. Direct Builder Method (`.Validate(...)`)

```go
act := action.New("order.process", handleOrder).
    Validate(func(ctx context.Context, req OrderReq) error {
        return validation.Struct(ctx, req)
    }).
    Build()
```

### 3. Pipeline Plugin Hook (`validation.New()`)

```go
plugin := validation.New()

for _, act := range registry.Actions() {
    act.AddAnyHook(plugin.(action.Executable).Describe().Hooks[0])
}
```

### 4. Standalone Validation

```go
if err := validation.Struct(ctx, myStruct); err != nil {
    return validation.FromValidatorError(err)
}
```

## Extension Paattern

```go
package main

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/nexssp/validation"
)

// Register domain-specific rules at application cold-start
func init() {
	validation.RegisterCustom("nip", func(_ context.Context, fl validator.FieldLevel) bool {
		return isPolishNIPValid(fl.Field().String())
	})
}

type InvoiceDTO struct {
	TaxID string `json:"tax_id" validate:"required,nip"`
}

func isPolishNIPValid(nip string) bool {
	// App-level or domain-specific package implementation
	return true
}
```
