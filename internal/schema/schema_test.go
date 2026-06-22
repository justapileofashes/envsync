package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func loadSchema(t *testing.T, body string) *Schema {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.schema")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return s
}

func TestRequiredMissing(t *testing.T) {
	s := loadSchema(t, "DATABASE_URL required\n")
	v := s.Validate(map[string]string{})
	if len(v) != 1 || v[0].Key != "DATABASE_URL" {
		t.Fatalf("expected DATABASE_URL violation, got %+v", v)
	}
}

func TestTypoSquashing(t *testing.T) {
	s := loadSchema(t, "DATABASE_URL required\n")
	// Junior dev misspelled the key.
	v := s.Validate(map[string]string{"DATABSE_URL": "postgres://x"})
	if len(v) != 1 || v[0].Suggestion != "DATABSE_URL" {
		t.Fatalf("expected typo suggestion DATABSE_URL, got %+v", v)
	}
}

func TestPrefixViolation(t *testing.T) {
	s := loadSchema(t, "STRIPE_KEY required prefix=sk_\n")
	if v := s.Validate(map[string]string{"STRIPE_KEY": "pk_live_123"}); len(v) != 1 {
		t.Fatalf("expected prefix violation, got %+v", v)
	}
	if v := s.Validate(map[string]string{"STRIPE_KEY": "sk_live_123"}); len(v) != 0 {
		t.Fatalf("expected no violation, got %+v", v)
	}
}

func TestRegexAndOptional(t *testing.T) {
	s := loadSchema(t, "PORT optional regex=^[0-9]+$\n")
	if v := s.Validate(map[string]string{}); len(v) != 0 {
		t.Fatalf("optional missing should pass, got %+v", v)
	}
	if v := s.Validate(map[string]string{"PORT": "abc"}); len(v) != 1 {
		t.Fatalf("expected regex violation, got %+v", v)
	}
	if v := s.Validate(map[string]string{"PORT": "8080"}); len(v) != 0 {
		t.Fatalf("expected pass, got %+v", v)
	}
}

func TestAllValid(t *testing.T) {
	s := loadSchema(t, "# comment\nDATABASE_URL required\nSTRIPE_KEY required prefix=sk_\n")
	v := s.Validate(map[string]string{
		"DATABASE_URL": "postgres://x",
		"STRIPE_KEY":   "sk_test_1",
	})
	if len(v) != 0 {
		t.Fatalf("expected no violations, got %+v", v)
	}
}
