package validation_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xerr"
	"github.com/nexssp/validation"
)

type semanticPayload struct {
	Code string `json:"code" validate:"required"`
}

func (s semanticPayload) Validate() error {
	if s.Code == "FORBIDDEN" {
		return errors.New("code is explicitly forbidden")
	}
	return nil
}

type unvalidatedPayload struct {
	RawData string
}

func TestPluginValidator_HotPathCOWCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	plugin := validation.New()
	// Get the hook from AnyAction via GetAnyHooks()
	pluginHook := plugin.GetAnyHooks()[0]

	actTagged := action.New("test.tagged", func(_ context.Context, req semanticPayload) (semanticPayload, error) {
		return req, nil
	}).AnyHook(pluginHook).Build()

	actUntagged := action.New("test.untagged", func(_ context.Context, req unvalidatedPayload) (unvalidatedPayload, error) {
		return req, nil
	}).AnyHook(pluginHook).Build()

	// 1. Tagged Action - Valid
	if _, err := actTagged.Do(ctx, semanticPayload{Code: "OK"}); err != nil {
		t.Fatalf("expected pass, got: %v", err)
	}

	// 2. Tagged Action - Tag Failure
	if _, err := actTagged.Do(ctx, semanticPayload{Code: ""}); err == nil {
		t.Fatal("expected tag validation error, got nil")
	}

	// 3. Tagged Action - Semantic Interface Failure
	_, err := actTagged.Do(ctx, semanticPayload{Code: "FORBIDDEN"})
	if err == nil {
		t.Fatal("expected semantic validation error, got nil")
	}

	var appErr *xerr.AppError
	if !errors.As(err, &appErr) || appErr.Kind != xerr.KindValidation {
		t.Fatalf("expected xerr.KindValidation, got: %v", err)
	}

	// 4. Untagged Action - Bypasses Validation with Zero Overhead
	for i := 0; i < 100; i++ {
		_, err := actUntagged.Do(ctx, unvalidatedPayload{RawData: fmt.Sprintf("data_%d", i)})
		if err != nil {
			t.Fatalf("expected zero-bypass pass, got: %v", err)
		}
	}
}
