package chromeimport

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCookieDecryptRoundTrip(t *testing.T) {
	key := DeriveKey("test-safe-storage-password")
	if len(key) != cookieKeyLen {
		t.Fatalf("derived key len = %d, want %d", len(key), cookieKeyLen)
	}
	plaintext := []byte("session=abc123; loggedIn=true")
	enc, err := EncryptCookie(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if string(enc[:3]) != "v10" {
		t.Errorf("missing v10 prefix")
	}
	dec, err := DecryptCookie(key, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(dec, plaintext) {
		t.Errorf("round-trip mismatch: %q != %q", dec, plaintext)
	}
}

func TestDecryptRejectsNonV10(t *testing.T) {
	key := DeriveKey("x")
	if _, err := DecryptCookie(key, []byte("plain value")); err == nil {
		t.Error("expected error for non-v10 value")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	a := DeriveKey("same")
	b := DeriveKey("same")
	if !bytes.Equal(a, b) {
		t.Error("DeriveKey must be deterministic")
	}
	if bytes.Equal(DeriveKey("a"), DeriveKey("b")) {
		t.Error("different passwords must derive different keys")
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	// Default profile with bookmarks + cookies.
	mkProfile(t, root, "Default", true, true, false)
	mkProfile(t, root, "Profile 1", false, true, true)
	// Local State display names.
	writeJSON(t, filepath.Join(root, "Local State"), map[string]any{
		"profile": map[string]any{"info_cache": map[string]any{
			"Default":   map[string]any{"name": "Personal"},
			"Profile 1": map[string]any{"name": "Work"},
		}},
	})

	profiles, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Fatalf("found %d profiles, want 2", len(profiles))
	}
	byDir := map[string]Profile{}
	for _, p := range profiles {
		byDir[p.Dir] = p
	}
	if byDir["Default"].Name != "Personal" || !byDir["Default"].HasCookies {
		t.Errorf("Default profile wrong: %+v", byDir["Default"])
	}
	if byDir["Profile 1"].Name != "Work" || !byDir["Profile 1"].HasHistory {
		t.Errorf("Profile 1 wrong: %+v", byDir["Profile 1"])
	}
}

func TestMergeBookmarks(t *testing.T) {
	src := t.TempDir()
	tgt := t.TempDir()
	writeJSON(t, filepath.Join(src, "Bookmarks"), map[string]any{
		"version": 1,
		"roots": map[string]any{
			"bookmark_bar": map[string]any{"type": "folder", "name": "Bookmarks Bar", "children": []any{
				map[string]any{"type": "url", "name": "GitHub", "url": "https://github.com"},
			}},
		},
	})

	count, err := MergeBookmarks(src, tgt)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if count != 1 {
		t.Errorf("imported %d urls, want 1", count)
	}
	// Re-sync must not duplicate the imported folder.
	if _, err := MergeBookmarks(src, tgt); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(tgt, "Bookmarks"))
	var bf bookmarkFile
	json.Unmarshal(data, &bf)
	importFolders := 0
	for _, c := range bf.Roots["bookmark_bar"].Children {
		if c.Name == "Imported from Chrome" {
			importFolders++
		}
	}
	if importFolders != 1 {
		t.Errorf("re-sync duplicated import folder: %d", importFolders)
	}
}

// helpers

func mkProfile(t *testing.T, root, dir string, cookies, bookmarks, history bool) {
	t.Helper()
	p := filepath.Join(root, dir)
	os.MkdirAll(p, 0o700)
	if cookies {
		os.WriteFile(filepath.Join(p, "Cookies"), []byte("x"), 0o600)
	}
	if bookmarks {
		os.WriteFile(filepath.Join(p, "Bookmarks"), []byte("{}"), 0o600)
	}
	if history {
		os.WriteFile(filepath.Join(p, "History"), []byte("x"), 0o600)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, _ := json.Marshal(v)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
