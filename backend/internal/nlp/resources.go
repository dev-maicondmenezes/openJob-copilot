package nlp

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/maicondmenezes/openjob/internal/nlp/providers"
)

//go:embed assets
var embeddedAssets embed.FS

// Manifest describes the bundled linguistic resources. The version is part of
// every deterministic result so outputs can be reproduced later.
type Manifest struct {
	Version    string            `json:"version"`
	Stopwords  map[string]string `json:"stopwords"`
	Algorithms map[string]string `json:"algorithms"`
	Licenses   []string          `json:"licenses"`
}

// Resources holds the loaded linguistic resources and their provenance.
type Resources struct {
	manifest  Manifest
	stopwords map[Language]map[string]struct{}
	checksums map[string]string
}

var (
	defaultResourcesOnce sync.Once
	defaultResources     *Resources
	defaultResourcesErr  error
)

// EmbeddedResources returns the process-wide resources compiled into the
// binary. Loading is lazy and cached.
func EmbeddedResources() (*Resources, error) {
	defaultResourcesOnce.Do(func() {
		root, err := embeddedRoot()
		if err != nil {
			defaultResourcesErr = err
			return
		}
		defaultResources, defaultResourcesErr = LoadResources(providers.NewEmbedded(root))
	})
	return defaultResources, defaultResourcesErr
}

func embeddedRoot() (fs.FS, error) {
	return fs.Sub(embeddedAssets, "assets")
}

// LoadResources reads the manifest and every stopword list from provider.
func LoadResources(provider providers.Provider) (*Resources, error) {
	if provider == nil {
		return nil, fmt.Errorf("nlp: nil resource provider")
	}
	rawManifest, err := provider.ReadFile("manifest.json")
	if err != nil {
		return nil, fmt.Errorf("nlp: read manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return nil, fmt.Errorf("nlp: parse manifest: %w", err)
	}
	if manifest.Version == "" {
		return nil, fmt.Errorf("nlp: manifest without version")
	}

	res := &Resources{
		manifest:  manifest,
		stopwords: make(map[Language]map[string]struct{}, len(manifest.Stopwords)),
		checksums: make(map[string]string, len(manifest.Stopwords)),
	}
	for code, path := range manifest.Stopwords {
		lang := ParseLanguage(code)
		data, err := provider.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("nlp: read stopwords %s: %w", path, err)
		}
		res.stopwords[lang] = parseWordList(data)
		sum := sha256.Sum256(data)
		res.checksums[path] = hex.EncodeToString(sum[:])
	}
	return res, nil
}

// Version returns the bundled resource version.
func (r *Resources) Version() string {
	if r == nil {
		return ""
	}
	return r.manifest.Version
}

// Algorithms exposes the versioned algorithm identifiers.
func (r *Resources) Algorithms() map[string]string {
	if r == nil {
		return nil
	}
	out := make(map[string]string, len(r.manifest.Algorithms))
	for k, v := range r.manifest.Algorithms {
		out[k] = v
	}
	return out
}

// Checksums returns the sha256 of each bundled stopword file.
func (r *Resources) Checksums() map[string]string {
	if r == nil {
		return nil
	}
	out := make(map[string]string, len(r.checksums))
	for k, v := range r.checksums {
		out[k] = v
	}
	return out
}

// Stopwords returns the stopword set of a language. The returned map must be
// treated as read-only.
func (r *Resources) Stopwords(lang Language) map[string]struct{} {
	if r == nil {
		return nil
	}
	return r.stopwords[lang]
}

func parseWordList(data []byte) map[string]struct{} {
	set := make(map[string]struct{})
	for _, line := range strings.Split(string(data), "\n") {
		word := NormalizeWord(line)
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		set[word] = struct{}{}
	}
	return set
}
