// Package providers supplies the external linguistic resources consumed by
// the NLP core. Resources are always read from a local provider: the system
// never downloads stopword lists or stemmer data during initialization.
package providers

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Provider is a read-only source of resource files (stopwords, licenses,
// manifests). Paths are slash-separated and relative to the provider root.
type Provider interface {
	ReadFile(name string) ([]byte, error)
	List(prefix string) ([]string, error)
}

// Embedded serves resources compiled into the binary via go:embed.
type Embedded struct {
	FS fs.FS
}

// NewEmbedded wraps an fs.FS (typically the package embed.FS).
func NewEmbedded(fsys fs.FS) *Embedded { return &Embedded{FS: fsys} }

func (e *Embedded) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(e.FS, filepath.ToSlash(name))
}

func (e *Embedded) List(prefix string) ([]string, error) {
	prefix = strings.TrimSuffix(filepath.ToSlash(prefix), "/")
	var out []string
	err := fs.WalkDir(e.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if prefix == "" || strings.HasPrefix(path, prefix+"/") || path == prefix {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// Directory serves resources from a local directory, allowing operators to
// override the embedded snapshots without rebuilding the binary.
type Directory struct {
	Root string
}

// NewDirectory returns a provider rooted at dir.
func NewDirectory(dir string) *Directory { return &Directory{Root: dir} }

func (d *Directory) ReadFile(name string) ([]byte, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return nil, fmt.Errorf("providers: path escapes root: %q", name)
	}
	return os.ReadFile(filepath.Join(d.Root, clean))
}

func (d *Directory) List(prefix string) ([]string, error) {
	root := filepath.Join(d.Root, filepath.FromSlash(prefix))
	var out []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(d.Root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
