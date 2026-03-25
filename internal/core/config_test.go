package core

import "testing"

func TestNormalizeConfigAppliesDefaults(t *testing.T) {
	cfg, err := NormalizeConfig(Config{})
	if err != nil {
		t.Fatalf("NormalizeConfig() error = %v", err)
	}

	if cfg.Mode != ModeRelaxed {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, ModeRelaxed)
	}
	if cfg.TargetWidth != 160 || cfg.TargetHeight != 144 {
		t.Fatalf("Target = %dx%d, want 160x144", cfg.TargetWidth, cfg.TargetHeight)
	}
	if cfg.PalettePreset != "gbc-olive" {
		t.Fatalf("PalettePreset = %q, want gbc-olive", cfg.PalettePreset)
	}
	if cfg.PreviewScale != 6 {
		t.Fatalf("PreviewScale = %d, want 6", cfg.PreviewScale)
	}
}

func TestValidateConfigRejectsBadMode(t *testing.T) {
	err := ValidateConfig(DefaultConfig())
	if err != nil {
		t.Fatalf("ValidateConfig(default) error = %v", err)
	}

	if err := ValidateConfig(Config{Mode: "weird"}); err == nil {
		t.Fatal("ValidateConfig() error = nil, want error")
	}
}
