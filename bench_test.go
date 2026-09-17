package validation_test

import (
	"context"
	"testing"

	"github.com/nexssp/kernel/action"
	"github.com/nexssp/validation"
)

var (
	benchCtx          = context.Background()
	benchValidEmail   = "alice@nexss.com"
	benchInvalidEmail = "not-an-email"
)

// Hook on a request type with no tags: OnBuild drops the hook, Do() pays nothing.
func BenchmarkHook_UntaggedHotPath(b *testing.B) {
	act := action.New("bench.untagged", func(_ context.Context, r untaggedReq) (untaggedReq, error) {
		return r, nil
	}).AnyHook(validation.Hook()).Build()
	req := untaggedReq{Value: "x"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := act.Do(benchCtx, req); err != nil {
			b.Fatal(err)
		}
	}
}

// Hook on a tagged request with valid payload: full validation cost.
func BenchmarkHook_TaggedValid(b *testing.B) {
	act := action.New("bench.tagged.valid", func(_ context.Context, r taggedReq) (taggedReq, error) {
		return r, nil
	}).AnyHook(validation.Hook()).Build()
	req := taggedReq{Email: benchValidEmail}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := act.Do(benchCtx, req); err != nil {
			b.Fatal(err)
		}
	}
}

// Hook on a tagged request with invalid payload: error path + FromValidatorError.
func BenchmarkHook_TaggedInvalid(b *testing.B) {
	act := action.New("bench.tagged.invalid", func(_ context.Context, r taggedReq) (taggedReq, error) {
		return r, nil
	}).AnyHook(validation.Hook()).Build()
	req := taggedReq{Email: benchInvalidEmail}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = act.Do(benchCtx, req)
	}
}

func BenchmarkStruct_Valid(b *testing.B) {
	u := userDTO{Name: "Alice", Email: benchValidEmail}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = validation.Struct(benchCtx, u)
	}
}

func BenchmarkStruct_Invalid(b *testing.B) {
	u := userDTO{Name: "Al", Email: benchInvalidEmail}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = validation.Struct(benchCtx, u)
	}
}

func BenchmarkTrimStringFields_Flat(b *testing.B) {
	type flat struct {
		A string
		B string
		C string
	}
	v := &flat{A: "  a  ", B: " b ", C: " c "}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		validation.TrimStringFields(v)
	}
}

func BenchmarkTrimStringFields_Nested(b *testing.B) {
	type inner struct{ Name, City string }
	type outer struct {
		Primary  *inner
		Children []inner
	}
	v := &outer{
		Primary:  &inner{Name: " a ", City: " b "},
		Children: []inner{{" x ", " y "}, {" z ", " w "}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		validation.TrimStringFields(v)
	}
}

// FromValidatorError on an already-produced ValidationErrors value:
// pure mapping cost, no validator execution.
func BenchmarkFromValidatorError(b *testing.B) {
	rawErr := validation.Struct(benchCtx, userDTO{Name: "Al", Email: benchInvalidEmail})
	if rawErr == nil {
		b.Fatal("expected validation error")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = validation.FromValidatorError(rawErr)
	}
}

// Full pipeline: identity action + builder Validate + handler.
func BenchmarkValidateAction_Do(b *testing.B) {
	act := validation.ValidateAction[userDTO]("bench.validate_action")
	req := userDTO{Name: "Alice", Email: benchValidEmail}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := act.Do(benchCtx, req); err != nil {
			b.Fatal(err)
		}
	}
}

// Same as above but through AutoValidate, to compare wiring paths.
func BenchmarkAutoValidate_Do(b *testing.B) {
	act := validation.AutoValidate(
		action.New("bench.autovalidate", func(_ context.Context, r userDTO) (userDTO, error) {
			return r, nil
		}),
	).Build()
	req := userDTO{Name: "Alice", Email: benchValidEmail}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := act.Do(benchCtx, req); err != nil {
			b.Fatal(err)
		}
	}
}

// ── Alloc budgets ───────────────────────────────────────────────────────────
//
// Alloc counts are hardware-independent and act as regression guards.
// ns/op is intentionally not asserted: it varies 2-5× across CPUs and CI
// runners, producing false failures. Compare ns manually via `go test -bench`.

func TestAllocBudget_Hook_UntaggedHotPath(t *testing.T) {
	act := action.New("budget.untagged", func(_ context.Context, r untaggedReq) (untaggedReq, error) {
		return r, nil
	}).AnyHook(validation.Hook()).Build()
	req := untaggedReq{Value: "x"}

	allocs := testing.AllocsPerRun(10_000, func() {
		if _, err := act.Do(benchCtx, req); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("untagged hot path must be zero-alloc, got %.2f — Hook filter broken?", allocs)
	}
}

func TestAllocBudget_TrimStringFields_Flat(t *testing.T) {
	type flat struct{ A, B, C string }
	v := &flat{A: "  a  ", B: " b ", C: " c "}

	allocs := testing.AllocsPerRun(10_000, func() {
		validation.TrimStringFields(v)
	})
	if allocs != 0 {
		t.Fatalf("flat trim must be zero-alloc, got %.2f", allocs)
	}
}

func TestAllocBudget_TrimStringFields_Nested(t *testing.T) {
	type inner struct{ Name, City string }
	type outer struct {
		Primary  *inner
		Children []inner
	}
	v := &outer{
		Primary:  &inner{Name: " a ", City: " b "},
		Children: []inner{{" x ", " y "}, {" z ", " w "}},
	}

	allocs := testing.AllocsPerRun(10_000, func() {
		validation.TrimStringFields(v)
	})
	if allocs != 0 {
		t.Fatalf("nested trim must be zero-alloc, got %.2f", allocs)
	}
}

func TestAllocBudget_Hook_TaggedValid(t *testing.T) {
	act := action.New("budget.tagged.valid", func(_ context.Context, r taggedReq) (taggedReq, error) {
		return r, nil
	}).AnyHook(validation.Hook()).Build()
	req := taggedReq{Email: benchValidEmail}

	allocs := testing.AllocsPerRun(1_000, func() {
		if _, err := act.Do(benchCtx, req); err != nil {
			t.Fatal(err)
		}
	})
	// Measured ~6; budget 12 gives 2× headroom for minor refactors.
	if allocs > 12 {
		t.Fatalf("tagged valid budget exceeded: %.2f allocs (was ~6)", allocs)
	}
}

func TestAllocBudget_Hook_TaggedInvalid(t *testing.T) {
	act := action.New("budget.tagged.invalid", func(_ context.Context, r taggedReq) (taggedReq, error) {
		return r, nil
	}).AnyHook(validation.Hook()).Build()
	req := taggedReq{Email: benchInvalidEmail}

	allocs := testing.AllocsPerRun(1_000, func() {
		_, _ = act.Do(benchCtx, req)
	})
	// Measured ~31; budget 60 catches a 2× regression without churn.
	if allocs > 60 {
		t.Fatalf("tagged invalid budget exceeded: %.2f allocs (was ~31)", allocs)
	}
}

func TestAllocBudget_Struct_Valid(t *testing.T) {
	u := userDTO{Name: "Alice", Email: benchValidEmail}

	allocs := testing.AllocsPerRun(1_000, func() {
		_ = validation.Struct(benchCtx, u)
	})
	// Measured ~6; validator/v10 reflection has an irreducible floor.
	if allocs > 12 {
		t.Fatalf("Struct valid budget exceeded: %.2f allocs (was ~6)", allocs)
	}
}

func TestAllocBudget_FromValidatorError(t *testing.T) {
	rawErr := validation.Struct(benchCtx, userDTO{Name: "Al", Email: benchInvalidEmail})
	if rawErr == nil {
		t.Fatal("expected validation error")
	}

	allocs := testing.AllocsPerRun(1_000, func() {
		_ = validation.FromValidatorError(rawErr)
	})
	// Measured ~9; budget 20 catches a 2× regression.
	if allocs > 20 {
		t.Fatalf("FromValidatorError budget exceeded: %.2f allocs (was ~9)", allocs)
	}
}
