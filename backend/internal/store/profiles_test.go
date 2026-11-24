package store

import (
	"context"
	"errors"
	"testing"
)

func mustConfig(t *testing.T, base, model, sealed string) []byte {
	t.Helper()
	raw, err := EncodeProfileConfig(ProfileConfig{BaseURL: base, Model: model, SealedKey: sealed})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestProfilesCRUDAndActivation(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	if err := s.CreateProfile(ctx, "a", "A", mustConfig(t, "https://a.example/v1", "ma", "sealed-a"), false); err != nil {
		t.Fatalf("create a: %v", err)
	}
	if err := s.CreateProfile(ctx, "b", "B", mustConfig(t, "https://b.example/v1", "mb", "sealed-b"), true); err != nil {
		t.Fatalf("create b (active): %v", err)
	}
	// Two active profiles must be impossible (unique partial index).
	if err := s.CreateProfile(ctx, "c", "C", mustConfig(t, "https://c.example/v1", "mc", "sealed-c"), true); err == nil {
		t.Fatal("second active profile must be rejected")
	}

	list, err := s.ListProfiles(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %d profiles, err %v", len(list), err)
	}
	if list[0].ID != "b" || !list[0].IsDefault {
		t.Fatalf("active profile must come first, got %+v", list[0])
	}

	if err := s.ActivateProfile(ctx, "a"); err != nil {
		t.Fatalf("activate a: %v", err)
	}
	// The flat keys read by tailoring/scoring now mirror profile a.
	for key, want := range map[string]string{
		"llm.api_key": "sealed-a", "llm.base_url": "https://a.example/v1", "llm.model": "ma",
	} {
		if got, _ := s.GetSetting(ctx, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	list, _ = s.ListProfiles(ctx)
	if !list[0].IsDefault || list[0].ID != "a" || list[1].IsDefault {
		t.Fatalf("exactly profile a must be active: %+v", list)
	}

	if err := s.UpdateProfile(ctx, "a", "A2", mustConfig(t, "https://a2.example/v1", "ma2", "sealed-a2")); err != nil {
		t.Fatal(err)
	}
	p, err := s.GetProfile(ctx, "a")
	if err != nil || p.Name != "A2" || !p.IsDefault {
		t.Fatalf("update must keep the active flag: %+v err %v", p, err)
	}
	cfg, err := p.DecodeConfig()
	if err != nil || cfg.Model != "ma2" {
		t.Fatalf("decode: %+v err %v", cfg, err)
	}

	if err := s.DeleteProfile(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetProfile(ctx, "b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted profile err = %v, want ErrNotFound", err)
	}
}

func TestProfilesNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	if _, err := s.GetProfile(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get: %v", err)
	}
	if err := s.UpdateProfile(ctx, "nope", "x", []byte(`{}`)); !errors.Is(err, ErrNotFound) {
		t.Errorf("update: %v", err)
	}
	if err := s.DeleteProfile(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete: %v", err)
	}
	if err := s.ActivateProfile(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("activate: %v", err)
	}
}

// Regression: the first F4 draft reset every default before discovering the
// target did not exist, leaving no active profile.
func TestActivateMissingProfileKeepsCurrentActive(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	if err := s.CreateProfile(ctx, "a", "A", mustConfig(t, "https://a.example/v1", "ma", "k"), true); err != nil {
		t.Fatal(err)
	}
	if err := s.ActivateProfile(ctx, "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	p, _ := s.GetProfile(ctx, "a")
	if !p.IsDefault {
		t.Fatal("failed activation must not deactivate the current profile")
	}
}

func TestMigrateFromFlatConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("fresh install creates nothing", func(t *testing.T) {
		s := openTestStore(t)
		if err := s.MigrateFromFlatConfig(ctx); err != nil {
			t.Fatal(err)
		}
		if list, _ := s.ListProfiles(ctx); len(list) != 0 {
			t.Fatalf("got %d profiles, want 0 (onboarding empty state)", len(list))
		}
	})

	t.Run("legacy flat config becomes the first active profile", func(t *testing.T) {
		s := openTestStore(t)
		_ = s.SetSetting(ctx, "llm.api_key", "sealed-legacy")
		_ = s.SetSetting(ctx, "llm.base_url", "https://api.groq.com/openai/v1")
		_ = s.SetSetting(ctx, "llm.model", "llama-3.3-70b-versatile")

		for i := 0; i < 2; i++ { // second run must be a no-op
			if err := s.MigrateFromFlatConfig(ctx); err != nil {
				t.Fatal(err)
			}
		}
		list, _ := s.ListProfiles(ctx)
		if len(list) != 1 || !list[0].IsDefault {
			t.Fatalf("want one active profile, got %+v", list)
		}
		cfg, _ := list[0].DecodeConfig()
		if cfg.SealedKey != "sealed-legacy" || cfg.Model != "llama-3.3-70b-versatile" ||
			cfg.BaseURL != "https://api.groq.com/openai/v1" {
			t.Fatalf("legacy values lost: %+v", cfg)
		}
	})
}
