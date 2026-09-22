package keepass

import (
	"os"
	"path/filepath"
	"testing"

	gokeepasslib "github.com/tobischo/gokeepasslib/v3"
)

func createTestKDBX(t *testing.T, pass string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "test.kdbx")
	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer f.Close()

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(pass)
	mkEntry := func(user, pw, note string) gokeepasslib.Entry {
		e := gokeepasslib.NewEntry()
		e.Values = append(e.Values,
			gokeepasslib.ValueData{Key: "UserName", Value: gokeepasslib.V{Content: user}},
			gokeepasslib.ValueData{Key: "Password", Value: gokeepasslib.V{Content: pw}},
			gokeepasslib.ValueData{Key: "Notes", Value: gokeepasslib.V{Content: note}},
		)
		return e
	}
	g1 := gokeepasslib.NewGroup()
	g1.Entries = []gokeepasslib.Entry{mkEntry("user1", "pass1", "marker note")}
	g2 := gokeepasslib.NewGroup()
	g2.Entries = []gokeepasslib.Entry{mkEntry("user2", "pass2", "other note")}
	inner := gokeepasslib.NewGroup()
	inner.Entries = []gokeepasslib.Entry{mkEntry("deep", "deep-pass", "marker note")}
	g2.Groups = []gokeepasslib.Group{inner}
	db.Content.Root.Groups = append(db.Content.Root.Groups, g1, g2)

	if err := gokeepasslib.NewEncoder(f).Encode(db); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return file
}

func TestLoad(t *testing.T) {
	file := createTestKDBX(t, "secret")
	var items []NPItem
	if err := Load(file, "secret", "marker note", &items); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2 (incl. entry from nested group)", len(items))
	}
	passes := make(map[string]string, len(items))
	for _, it := range items {
		passes[it.Name] = it.Pass
	}
	if passes["user1"] != "pass1" {
		t.Errorf("user1 pass = %q, want pass1", passes["user1"])
	}
	if passes["deep"] != "deep-pass" {
		t.Errorf("deep pass = %q, want deep-pass", passes["deep"])
	}
}

func TestLoadErrors(t *testing.T) {
	file := createTestKDBX(t, "secret")
	var items []NPItem
	if err := Load(file, "wrong-password", "marker", &items); err == nil {
		t.Error("expected error for wrong password")
	}
	if err := Load(filepath.Join(t.TempDir(), "missing.kdbx"), "x", "n", &items); err == nil {
		t.Error("expected error for missing file")
	}
}
