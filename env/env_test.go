package env

import (
	"os"
	"testing"
)

// ---- Вспомогательные структуры для тестов ----

type testBasic struct {
	Host string `env:"HOST" default:"localhost"`
	Port int    `env:"PORT" default:"8080"`
}

type testNoDefault struct {
	Key string `env:"KEY"`
}

type testMixedTypes struct {
	Name   string  `env:"NAME" default:"default_name"`
	Count  int     `env:"COUNT" default:"42"`
	Rate   float64 `env:"RATE" default:"3.14"`
	Active bool    `env:"ACTIVE" default:"true"`
	Uport  uint    `env:"UPORT" default:"9000"`
}

type testUnexported struct {
	Exported   string `env:"EXPORTED" default:"yes"`
	unexported string `env:"UNEXPORTED" default:"no"`
}

type testNoEnvTag struct {
	SkipMe string `default:"skip"`
}

type testNotStruct int

// ---- setup/teardown ----

func setupEnv(vars map[string]string) func() {
	prev := make(map[string]string)
	for k, v := range vars {
		prev[k] = os.Getenv(k)
		os.Setenv(k, v)
	}
	return func() {
		for k := range vars {
			if prev[k] == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, prev[k])
			}
		}
	}
}

// ---- Тесты ----

func TestLoad_BasicDefaults(t *testing.T) {
	cfg := testBasic{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	cleanup := setupEnv(map[string]string{
		"HOST": "example.com",
		"PORT": "3000",
	})
	defer cleanup()

	cfg := testBasic{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "example.com")
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 3000)
	}
}

func TestLoad_EnvOverridesDefault(t *testing.T) {
	cleanup := setupEnv(map[string]string{
		"PORT": "9999",
	})
	defer cleanup()

	cfg := testBasic{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// HOST берётся из default, PORT из env
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 9999 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9999)
	}
}

func TestLoad_MissingFieldWithoutDefault(t *testing.T) {
	cfg := testNoDefault{}
	err := Load(&cfg)
	if err == nil {
		t.Fatal("expected error for missing field without default")
	}
}

func TestLoad_NotAPointer(t *testing.T) {
	cfg := testBasic{}
	err := Load(cfg) // передаём value, а не pointer
	if err == nil {
		t.Fatal("expected error for non-pointer")
	}
}

func TestLoad_PointerToNotStruct(t *testing.T) {
	var x testNotStruct
	err := Load(&x)
	if err == nil {
		t.Fatal("expected error for pointer to non-struct")
	}
}

func TestLoad_MixedTypes(t *testing.T) {
	cfg := testMixedTypes{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "default_name" {
		t.Errorf("Name = %q, want %q", cfg.Name, "default_name")
	}
	if cfg.Count != 42 {
		t.Errorf("Count = %d, want %d", cfg.Count, 42)
	}
	if cfg.Rate != 3.14 {
		t.Errorf("Rate = %f, want %f", cfg.Rate, 3.14)
	}
	if cfg.Active != true {
		t.Errorf("Active = %v, want %v", cfg.Active, true)
	}
	if cfg.Uport != 9000 {
		t.Errorf("Uport = %d, want %d", cfg.Uport, 9000)
	}
}

func TestLoad_MixedTypesFromEnv(t *testing.T) {
	cleanup := setupEnv(map[string]string{
		"NAME":   "from_env",
		"COUNT":  "777",
		"RATE":   "2.718",
		"ACTIVE": "false",
		"UPORT":  "65535",
	})
	defer cleanup()

	cfg := testMixedTypes{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "from_env" {
		t.Errorf("Name = %q, want %q", cfg.Name, "from_env")
	}
	if cfg.Count != 777 {
		t.Errorf("Count = %d, want %d", cfg.Count, 777)
	}
	if cfg.Rate != 2.718 {
		t.Errorf("Rate = %f, want %f", cfg.Rate, 2.718)
	}
	if cfg.Active != false {
		t.Errorf("Active = %v, want %v", cfg.Active, false)
	}
	if cfg.Uport != 65535 {
		t.Errorf("Uport = %d, want %d", cfg.Uport, 65535)
	}
}

func TestLoad_InvalidInt(t *testing.T) {
	cleanup := setupEnv(map[string]string{
		"PORT": "not_a_number",
	})
	defer cleanup()

	cfg := testBasic{}
	err := Load(&cfg)
	if err == nil {
		t.Fatal("expected error for invalid int value")
	}
}

func TestLoad_InvalidBool(t *testing.T) {
	type cfg struct {
		Flag bool `env:"FLAG" default:"not_a_bool"`
	}
	c := cfg{}
	err := Load(&c)
	if err == nil {
		t.Fatal("expected error for invalid bool default")
	}
}

func TestLoad_InvalidFloat(t *testing.T) {
	type cfg struct {
		Rate float64 `env:"RATE" default:"not_a_float"`
	}
	c := cfg{}
	err := Load(&c)
	if err == nil {
		t.Fatal("expected error for invalid float default")
	}
}

func TestLoad_UnexportedField(t *testing.T) {
	cleanup := setupEnv(map[string]string{
		"EXPORTED": "from_env",
	})
	defer cleanup()

	cfg := testUnexported{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Exported != "from_env" {
		t.Errorf("Exported = %q, want %q", cfg.Exported, "from_env")
	}
	// unexported поле должно остаться с zero value
	if cfg.unexported != "" {
		t.Errorf("unexported = %q, want %q", cfg.unexported, "")
	}
}

func TestLoad_FieldWithoutEnvTag(t *testing.T) {
	cfg := testNoEnvTag{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Поле без env-тега должно игнорироваться, остаётся zero value
	if cfg.SkipMe != "" {
		t.Errorf("SkipMe = %q, want %q", cfg.SkipMe, "")
	}
}

func TestLoad_EmptyEnvStringFallsBackToDefault(t *testing.T) {
	cleanup := setupEnv(map[string]string{
		"HOST": "",
	})
	defer cleanup()

	cfg := testBasic{}
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Пустая строка в env = os.Getenv вернёт "", Load воспримет как "нет значения"
	// и подставит default
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want default %q", cfg.Host, "localhost")
	}
}

func TestLoad_NilPointer(t *testing.T) {
	err := Load(nil)
	if err == nil {
		t.Fatal("expected error for nil")
	}
}