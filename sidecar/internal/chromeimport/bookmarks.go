package chromeimport

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// MergeBookmarks merges the source profile's Bookmarks JSON into the target
// profile, appending source roots' children under an "Imported from Chrome"
// folder in the target bookmark bar (idempotent by folder name — re-sync won't
// duplicate the folder, it replaces it).
func MergeBookmarks(sourcePath, targetPath string) (int, error) {
	srcData, err := os.ReadFile(filepath.Join(sourcePath, "Bookmarks"))
	if err != nil {
		return 0, err
	}
	var src bookmarkFile
	if err := json.Unmarshal(srcData, &src); err != nil {
		return 0, err
	}

	target := bookmarkFile{Version: 1, Roots: map[string]bookmarkNode{}}
	if data, err := os.ReadFile(filepath.Join(targetPath, "Bookmarks")); err == nil {
		_ = json.Unmarshal(data, &target)
	}
	if target.Roots == nil {
		target.Roots = map[string]bookmarkNode{}
	}

	imported := collectChildren(src.Roots["bookmark_bar"])
	imported = append(imported, collectChildren(src.Roots["other"])...)

	folder := bookmarkNode{Type: "folder", Name: "Imported from Chrome", Children: imported}
	bar := target.Roots["bookmark_bar"]
	bar.Type = "folder"
	if bar.Name == "" {
		bar.Name = "Bookmarks Bar"
	}
	bar.Children = replaceImportFolder(bar.Children, folder)
	target.Roots["bookmark_bar"] = bar

	out, err := json.MarshalIndent(target, "", "   ")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(targetPath, 0o700); err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(targetPath, "Bookmarks"), out, 0o600); err != nil {
		return 0, err
	}
	return countURLs(imported), nil
}

type bookmarkFile struct {
	Version int                     `json:"version"`
	Roots   map[string]bookmarkNode `json:"roots"`
}

type bookmarkNode struct {
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	URL      string         `json:"url,omitempty"`
	Children []bookmarkNode `json:"children,omitempty"`
}

func collectChildren(n bookmarkNode) []bookmarkNode {
	return n.Children
}

// replaceImportFolder removes any prior "Imported from Chrome" folder, then
// appends the fresh one (keeps re-sync idempotent).
func replaceImportFolder(children []bookmarkNode, folder bookmarkNode) []bookmarkNode {
	kept := children[:0:0]
	for _, c := range children {
		if c.Type == "folder" && c.Name == folder.Name {
			continue
		}
		kept = append(kept, c)
	}
	return append(kept, folder)
}

func countURLs(nodes []bookmarkNode) int {
	n := 0
	for _, node := range nodes {
		if node.Type == "url" {
			n++
		}
		n += countURLs(node.Children)
	}
	return n
}
