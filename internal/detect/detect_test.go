package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectNextByConfig(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "next.config.js", "module.exports = {}")
	got := Detect(dir)
	if got.Framework != "Next.js" || got.EnvFile != ".env.local" {
		t.Fatalf("got %+v, want Next.js/.env.local", got)
	}
}

func TestDetectViteByDep(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"devDependencies":{"vite":"^5.0.0"}}`)
	got := Detect(dir)
	if got.Framework != "Vite" || got.EnvFile != ".env.development" {
		t.Fatalf("got %+v, want Vite/.env.development", got)
	}
}

func TestDetectNextDepBeatsNothing(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"dependencies":{"next":"14.0.0","react":"18"}}`)
	got := Detect(dir)
	if got.Framework != "Next.js" {
		t.Fatalf("got %+v, want Next.js", got)
	}
}

func TestDetectUnknownFallsBack(t *testing.T) {
	dir := t.TempDir()
	got := Detect(dir)
	if got.Framework != "" || got.EnvFile != ".env" {
		t.Fatalf("got %+v, want empty/.env", got)
	}
}

func TestDetectMalformedPackageJSON(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{not valid json`)
	got := Detect(dir)
	if got.EnvFile != ".env" {
		t.Fatalf("malformed package.json should fall back to .env, got %+v", got)
	}
}
