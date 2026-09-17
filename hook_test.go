package validation_test

import (
	"context"
	"testing"

	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xerr"
	"github.com/nexssp/validation"
)

type taggedReq struct {
	Email string `json:"email" validate:"required,email"`
}

type untaggedReq struct {
	Value string
}

type nestedFilter struct {
	Keyword string `json:"keyword" validate:"required,min=2"`
}

type nestedParent struct {
	ID     string       `json:"id"`
	Filter nestedFilter `json:"filter"`
}

type recursiveTree struct {
	Value    string          `json:"value"`
	Children []recursiveTree `json:"children"`
}

type semanticReq struct {
	Code string `json:"code" validate:"required"`
}

func (s semanticReq) Validate() error {
	if s.Code == "FORBIDDEN" {
		return xerr.Validation("code is explicitly forbidden")
	}
	return nil
}

func TestHook_SkipsUntaggedAction(t *testing.T) {
	act := action.New("hook.skipped", func(_ context.Context, r untaggedReq) (untaggedReq, error) {
		return r, nil
	}).Build()

	act.AddAnyHook(validation.Hook())

	if got := len(act.GetAnyHooks()); got != 0 {
		t.Fatalf("expected 0 hooks for untagged Req, got %d", got)
	}
}

func TestHook_AppliesToTaggedAction(t *testing.T) {
	act := action.New("hook.applied", func(_ context.Context, r taggedReq) (taggedReq, error) {
		return r, nil
	}).Build()

	act.AddAnyHook(validation.Hook())

	if got := len(act.GetAnyHooks()); got != 1 {
		t.Fatalf("expected 1 hook for tagged Req, got %d", got)
	}

	_, err := act.Do(context.Background(), taggedReq{Email: "not-an-email"})
	if xerr.KindFrom(err) != xerr.KindValidation {
		t.Fatalf("expected KindValidation, got %v", err)
	}
}

func TestHook_ViaLibraryHooks(t *testing.T) {
	tagged := action.New("lib.tagged", func(_ context.Context, r taggedReq) (taggedReq, error) {
		return r, nil
	}).Build()
	untagged := action.New("lib.untagged", func(_ context.Context, r untaggedReq) (untaggedReq, error) {
		return r, nil
	}).Build()

	reg := action.MustNewRegistry(action.Library{
		Name:    "validation.test",
		Actions: []action.AnyAction{tagged, untagged},
		Hooks:   []action.AnyHook{validation.Hook()},
	})

	a, _ := reg.Get("lib.tagged")
	if got := len(a.GetAnyHooks()); got != 1 {
		t.Fatalf("tagged: expected 1 hook, got %d", got)
	}

	b, _ := reg.Get("lib.untagged")
	if got := len(b.GetAnyHooks()); got != 0 {
		t.Fatalf("untagged: expected 0 hooks, got %d", got)
	}
}

func TestHook_DetectsNestedTaggedStruct(t *testing.T) {
	act := action.New("hook.nested", func(_ context.Context, r nestedParent) (nestedParent, error) {
		return r, nil
	}).Build()

	act.AddAnyHook(validation.Hook())

	if got := len(act.GetAnyHooks()); got != 1 {
		t.Fatalf("nested tagged struct must accept, got %d hooks", got)
	}
}

func TestHook_RecursiveTypeTerminatesAndRejects(t *testing.T) {
	act := action.New("hook.recursive", func(_ context.Context, r recursiveTree) (recursiveTree, error) {
		return r, nil
	}).Build()

	// Must not hang or stack-overflow during OnBuild.
	act.AddAnyHook(validation.Hook())

	if got := len(act.GetAnyHooks()); got != 0 {
		t.Fatalf("recursive struct without tags must reject, got %d hooks", got)
	}
}

func TestHook_DetectsValidatableInterface(t *testing.T) {
	act := action.New("hook.semantic", func(_ context.Context, r semanticReq) (semanticReq, error) {
		return r, nil
	}).Build()

	act.AddAnyHook(validation.Hook())

	if got := len(act.GetAnyHooks()); got != 1 {
		t.Fatalf("Validatable implementer must accept, got %d hooks", got)
	}

	if _, err := act.Do(context.Background(), semanticReq{Code: "OK"}); err != nil {
		t.Fatalf("valid payload failed: %v", err)
	}

	_, err := act.Do(context.Background(), semanticReq{Code: "FORBIDDEN"})
	if xerr.KindFrom(err) != xerr.KindValidation {
		t.Fatalf("expected KindValidation, got %v", err)
	}
}

func TestHook_ZeroAllocForUntaggedHotPath(t *testing.T) {
	act := action.New("hook.hot", func(_ context.Context, r untaggedReq) (untaggedReq, error) {
		return r, nil
	}).Build()

	act.AddAnyHook(validation.Hook())

	payload := untaggedReq{Value: "x"}
	ctx := context.Background()

	allocs := testing.AllocsPerRun(10_000, func() {
		if _, err := act.Do(ctx, payload); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocs/op for untagged hot path, got %.2f", allocs)
	}
}
