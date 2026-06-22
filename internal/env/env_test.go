package env

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiffCategories(t *testing.T) {
	left := map[string]string{"A": "1", "B": "2", "C": "3"}
	right := map[string]string{"B": "2", "C": "different", "D": "4"}
	d := Diff(left, right)

	if !reflect.DeepEqual(d.OnlyLeft, []string{"A"}) {
		t.Errorf("OnlyLeft = %v, want [A]", d.OnlyLeft)
	}
	if !reflect.DeepEqual(d.OnlyRight, []string{"D"}) {
		t.Errorf("OnlyRight = %v, want [D]", d.OnlyRight)
	}
	if !reflect.DeepEqual(d.Changed, []string{"C"}) {
		t.Errorf("Changed = %v, want [C]", d.Changed)
	}
	if !reflect.DeepEqual(d.Same, []string{"B"}) {
		t.Errorf("Same = %v, want [B]", d.Same)
	}
	if d.Identical() {
		t.Error("Identical() should be false")
	}
}

func TestDiffIdentical(t *testing.T) {
	m := map[string]string{"X": "1"}
	if !Diff(m, map[string]string{"X": "1"}).Identical() {
		t.Error("expected identical")
	}
}

func TestMergeOverride(t *testing.T) {
	dir := t.TempDir()
	ovr := filepath.Join(dir, ".env.override")
	if err := os.WriteFile(ovr, []byte("DB_URL=local\nEXTRA=yes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &Payload{Version: 1, Values: map[string]string{"DB_URL": "remote", "KEEP": "1"}}
	n, err := p.MergeOverride(ovr)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("merged %d, want 2", n)
	}
	if p.Values["DB_URL"] != "local" {
		t.Errorf("override did not win: %q", p.Values["DB_URL"])
	}
	if p.Values["EXTRA"] != "yes" || p.Values["KEEP"] != "1" {
		t.Errorf("unexpected values: %+v", p.Values)
	}
}

func TestMergeOverrideMissingFileNoop(t *testing.T) {
	p := &Payload{Values: map[string]string{"A": "1"}}
	n, err := p.MergeOverride(filepath.Join(t.TempDir(), "nope"))
	if err != nil || n != 0 {
		t.Fatalf("missing override should be no-op, got n=%d err=%v", n, err)
	}
}
