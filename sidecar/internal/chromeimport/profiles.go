// Package chromeimport discovers a user's Chrome profiles and imports their
// data (bookmarks, decrypted cookies) into a workspace's browser profile so an
// agent can browse with the user's existing logins.
package chromeimport

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Profile describes a discovered Chrome profile and which data it has.
type Profile struct {
	Name         string `json:"name"`        // display name (from Local State), e.g. "Personal"
	Dir          string `json:"dir"`         // profile dir name, e.g. "Default" or "Profile 1"
	Path         string `json:"path"`        // absolute profile path
	HasCookies   bool   `json:"hasCookies"`
	HasBookmarks bool   `json:"hasBookmarks"`
	HasHistory   bool   `json:"hasHistory"`
}

// ChromeRoot returns the default Chrome user-data dir for the current user.
func ChromeRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "Google", "Chrome"), nil
}

// Scan enumerates profiles under root, reading display names from Local State.
func Scan(root string) ([]Profile, error) {
	names := profileDisplayNames(root)

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var profiles []Profile
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := e.Name()
		if dir != "Default" && !isProfileDir(dir) {
			continue
		}
		path := filepath.Join(root, dir)
		display := names[dir]
		if display == "" {
			display = dir
		}
		profiles = append(profiles, Profile{
			Name:         display,
			Dir:          dir,
			Path:         path,
			HasCookies:   exists(filepath.Join(path, "Cookies")),
			HasBookmarks: exists(filepath.Join(path, "Bookmarks")),
			HasHistory:   exists(filepath.Join(path, "History")),
		})
	}
	return profiles, nil
}

// profileDisplayNames reads Local State's profile.info_cache for friendly names.
func profileDisplayNames(root string) map[string]string {
	out := map[string]string{}
	data, err := os.ReadFile(filepath.Join(root, "Local State"))
	if err != nil {
		return out
	}
	var ls struct {
		Profile struct {
			InfoCache map[string]struct {
				Name string `json:"name"`
			} `json:"info_cache"`
		} `json:"profile"`
	}
	if json.Unmarshal(data, &ls) != nil {
		return out
	}
	for dir, info := range ls.Profile.InfoCache {
		out[dir] = info.Name
	}
	return out
}

func isProfileDir(name string) bool {
	const prefix = "Profile "
	return len(name) > len(prefix) && name[:len(prefix)] == prefix
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
