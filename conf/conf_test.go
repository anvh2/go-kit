package conf

import (
	"errors"
	"os"
	"slices"
	"testing"
	"time"
)

type source struct {
	Kind        string `mapstructure:"kind"`
	FixturePath string `mapstructure:"fixture_path"`
}

type kind string

type testConfig struct {
	Env     string        `mapstructure:"env"`
	Name    string        `mapstructure:"name"`
	Modules []string      `mapstructure:"modules"`
	Kinds   []kind        `mapstructure:"kinds"`
	Timeout time.Duration `mapstructure:"timeout"`
	Source  source        `mapstructure:"source"`
}

// inDir runs the test from a fresh directory holding the given config.yaml (none when empty).
func inDir(t *testing.T, yaml string) {
	t.Helper()
	dir := t.TempDir()
	if yaml != "" {
		if err := os.WriteFile(dir+"/config.yaml", []byte(yaml), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestLoadLayersDefaultsYAMLAndEnvironment(t *testing.T) {
	inDir(t, "name: from-yaml\nmodules: [a, b]\nsource:\n  kind: fixture\n  fixture_path: yaml.json\n")
	t.Setenv("SOURCE__FIXTURE_PATH", "env.json")
	t.Setenv("MODULES", "x,y")
	t.Setenv("TIMEOUT", "1m30s")
	t.Setenv("NAME", "") // empty counts as unset

	cfg := testConfig{Env: "dev", Timeout: time.Second}
	if err := Load(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "dev" || cfg.Name != "from-yaml" || cfg.Source.Kind != "fixture" || cfg.Source.FixturePath != "env.json" ||
		!slices.Equal(cfg.Modules, []string{"x", "y"}) || cfg.Timeout != 90*time.Second {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadReadsSettingsMissingFromYAMLFromTheEnvironment(t *testing.T) {
	inDir(t, "")
	t.Setenv("SOURCE__KIND", "api")
	t.Setenv("MODULES", `["j","k"]`)
	t.Setenv("KINDS", "p,q") // a list of a named string type splits too
	cfg := testConfig{}
	if err := Load(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Source.Kind != "api" || !slices.Equal(cfg.Modules, []string{"j", "k"}) || !slices.Equal(cfg.Kinds, []kind{"p", "q"}) {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestStrictRejectsUnknownKeysByName(t *testing.T) {
	inDir(t, "nmae: typo\nsource:\n  kindd: x\n")
	err := Load(&testConfig{}, Strict())
	var unknown *UnknownKeysError
	if !errors.As(err, &unknown) || !slices.Equal(unknown.Keys, []string{"nmae", "source.kindd"}) {
		t.Fatalf("want unknown keys nmae and source.kindd, got %v", err)
	}
	if Load(&testConfig{}) != nil {
		t.Error("without Strict, unknown keys are ignored")
	}
}

func TestLoadRejectsMalformedYAML(t *testing.T) {
	inDir(t, "name: [unclosed\n")
	if err := Load(&testConfig{}); err == nil {
		t.Error("want an error for malformed config.yaml")
	}
}
