// notes/notes.go
package notes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Dir returns the notes directory, creating it if needed.
func Dir() (string, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "dovin", "notes")
	return dir, os.MkdirAll(dir, 0755)
}

// Filename returns a deterministic filename for a task.
func Filename(id int64, title string) string {
	slug := slugify(title)
	return fmt.Sprintf("%d-%s.md", id, slug)
}

// EnsureFile creates the notes file if it does not exist, then opens it.
// Returns the relative filename (for storage in notes_path).
func EnsureFile(id int64, title string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	name := Filename(id, title)
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("# "+title+"\n\n"), 0644); err != nil {
			return "", fmt.Errorf("create notes file: %w", err)
		}
	}
	if err := open(path); err != nil {
		return "", fmt.Errorf("open notes file: %w", err)
	}
	return name, nil
}

func open(path string) error {
	editor := os.Getenv("EDITOR")
	if editor != "" {
		return exec.Command(editor, path).Start()
	}
	return exec.Command("open", path).Start()
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
