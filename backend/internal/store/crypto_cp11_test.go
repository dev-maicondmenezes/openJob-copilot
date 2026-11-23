package store

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestMasterKeyCreateIsIdempotentAndPrivate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "data")
	path := filepath.Join(dir, MasterKeyFileName)

	k1, err := LoadOrCreateMasterKey(path)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := LoadOrCreateMasterKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(k1, k2) {
		t.Fatal("second run regenerated the master key")
	}

	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(path)
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("key mode = %v, want 0600", fi.Mode().Perm())
		}
		di, _ := os.Stat(dir)
		if di.Mode().Perm() != 0o700 {
			t.Errorf("new data dir mode = %v, want 0700", di.Mode().Perm())
		}
	}

	// No temp leftovers beside the key (atomic publish cleans up).
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the key in %s, got %d entries", dir, len(entries))
	}
}

func TestMasterKeyNeverOverwritesCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), MasterKeyFileName)
	if err := os.WriteFile(path, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateMasterKey(path); err == nil {
		t.Fatal("expected error for a truncated key")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "short" {
		t.Errorf("existing file was modified: %q", got)
	}
}

func TestMasterKeyConcurrentFirstRunConverges(t *testing.T) {
	path := filepath.Join(t.TempDir(), MasterKeyFileName)

	const n = 16
	keys := make([][]byte, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			keys[i], errs[i] = LoadOrCreateMasterKey(path)
		}()
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if !bytes.Equal(keys[i], keys[0]) {
			t.Fatalf("goroutine %d got a different key: racing first runs must converge", i)
		}
	}
	onDisk, _ := os.ReadFile(path)
	if !bytes.Equal(onDisk, keys[0]) {
		t.Error("returned key differs from the one on disk")
	}
}
