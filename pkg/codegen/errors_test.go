package codegen

import (
	"errors"
	"fmt"
	"testing"
)

func TestTemplateError(t *testing.T) {
	tests := []struct {
		name    string
		te      TemplateError
		contain string
	}{
		{
			name: "parse error with template name",
			te: TemplateError{
				Op:   "parse",
				Name: "overload.js.tmpl",
				Err:  fmt.Errorf("template: overload.js.tmpl:3: unexpected"),
			},
			contain: "overload.js.tmpl",
		},
		{
			name: "render error without name",
			te: TemplateError{
				Op:  "render",
				Err: fmt.Errorf("execution error"),
			},
			contain: "render",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.te.Error()
			if !contains(errStr, tt.contain) {
				t.Errorf("Error() = %q, want containing %q", errStr, tt.contain)
			}
		})
	}

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := fmt.Errorf("original error")
		te := &TemplateError{Op: "parse", Err: underlying}
		if !errors.Is(te, underlying) {
			t.Error("errors.Is should find underlying error via Unwrap()")
		}
	})
}

func TestGenerateError(t *testing.T) {
	t.Run("Error contains op and underlying message", func(t *testing.T) {
		ge := &GenerateError{
			Op:  "generate",
			Err: fmt.Errorf("empty hooks list"),
		}
		errStr := ge.Error()
		if !contains(errStr, "generate") {
			t.Errorf("Error() = %q, want containing %q", errStr, "generate")
		}
		if !contains(errStr, "empty hooks list") {
			t.Errorf("Error() = %q, want containing %q", errStr, "empty hooks list")
		}
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := fmt.Errorf("original error")
		ge := &GenerateError{Op: "generate", Err: underlying}
		if !errors.Is(ge, underlying) {
			t.Error("errors.Is should find underlying error via Unwrap()")
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
