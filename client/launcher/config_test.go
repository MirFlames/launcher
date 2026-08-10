package main

import "testing"

// withBuildDefaults подменяет ldflags-дефолты на время теста (в обычном
// `go test` они пустые, а вся логика профилей строится вокруг них).
func withBuildDefaults(t *testing.T, api, host, port string) {
	t.Helper()
	oldAPI, oldHost, oldPort := buildDefaultApiBaseUrl, buildDefaultServerHost, buildDefaultServerPort
	buildDefaultApiBaseUrl, buildDefaultServerHost, buildDefaultServerPort = api, host, port
	t.Cleanup(func() {
		buildDefaultApiBaseUrl, buildDefaultServerHost, buildDefaultServerPort = oldAPI, oldHost, oldPort
	})
}

func TestMigrateEnvProfilesKeepsProdWhenConfigMatchesBuild(t *testing.T) {
	withBuildDefaults(t, "https://prod.example", "prod.example", "25565")
	cfg := &Config{ApiBaseUrl: "https://prod.example", ServerHost: "prod.example", ServerPort: 25565}
	cfg.migrateEnvProfiles()

	if cfg.EnvName() != envProd {
		t.Fatalf("env = %q, ожидался %q", cfg.EnvName(), envProd)
	}
	if _, ok := cfg.EnvProfiles[envDev]; ok {
		t.Fatalf("dev-профиль не должен создаваться из прод-адресов: %+v", cfg.EnvProfiles)
	}
}

func TestMigrateEnvProfilesMovesOverridesToDev(t *testing.T) {
	withBuildDefaults(t, "https://prod.example", "prod.example", "25565")
	cfg := &Config{ApiBaseUrl: "http://localhost:80", ServerHost: "127.0.0.1", ServerPort: 25566}
	cfg.migrateEnvProfiles()

	if cfg.EnvName() != envDev {
		t.Fatalf("env = %q, ожидался %q", cfg.EnvName(), envDev)
	}
	if got := cfg.EnvProfiles[envDev]; got.ApiBaseUrl != "http://localhost:80" || got.ServerHost != "127.0.0.1" || got.ServerPort != 25566 {
		t.Fatalf("dev-профиль собран неверно: %+v", got)
	}
	// Эффективные поля миграция не трогает — установка работает как раньше.
	if cfg.ApiBaseUrl != "http://localhost:80" {
		t.Fatalf("эффективный apiBaseUrl изменён миграцией: %q", cfg.ApiBaseUrl)
	}
}

func TestMigrateEnvProfilesSkipsWhenProfilesExist(t *testing.T) {
	cfg := &Config{
		ApiBaseUrl:  "http://localhost:80",
		EnvProfiles: map[string]EnvProfile{envDev: {ApiBaseUrl: "http://stand"}},
	}
	cfg.migrateEnvProfiles()

	if got := cfg.EnvProfiles[envDev].ApiBaseUrl; got != "http://stand" {
		t.Fatalf("существующий профиль перезаписан: %q", got)
	}
}

func TestProfileProdFallsBackToBuildDefaults(t *testing.T) {
	withBuildDefaults(t, "https://prod.example", "prod.example", "25565")
	cfg := &Config{}

	prod := cfg.Profile(envProd)
	if prod.ApiBaseUrl != "https://prod.example" || prod.ServerHost != "prod.example" || prod.ServerPort != 25565 {
		t.Fatalf("прод-профиль не дозаполнен дефолтами сборки: %+v", prod)
	}
	// Для dev дозаполнения быть не должно — иначе стенд молча уедет на прод.
	if dev := cfg.Profile(envDev); dev != (EnvProfile{}) {
		t.Fatalf("dev-профиль дозаполнен дефолтами сборки: %+v", dev)
	}
}

func TestApplyProfileSwitchesEffectiveValues(t *testing.T) {
	withBuildDefaults(t, "https://prod.example", "prod.example", "25565")
	cfg := &Config{
		EnvProfiles: map[string]EnvProfile{
			envDev: {ApiBaseUrl: "http://localhost:80", ServerHost: "127.0.0.1", ServerPort: 25566},
		},
	}

	cfg.ApplyProfile(envDev)
	if cfg.Env != envDev || cfg.ApiBaseUrl != "http://localhost:80" || cfg.ServerHost != "127.0.0.1" || cfg.ServerPort != 25566 {
		t.Fatalf("переключение на dev не применилось: %+v", cfg)
	}

	cfg.ApplyProfile(envProd)
	if cfg.Env != envProd || cfg.ApiBaseUrl != "https://prod.example" || cfg.ServerHost != "prod.example" || cfg.ServerPort != 25565 {
		t.Fatalf("возврат на прод не применился: %+v", cfg)
	}
	// Dev-профиль пережил переключение — адреса стенда не надо вводить заново.
	if got := cfg.EnvProfiles[envDev]; got.ApiBaseUrl != "http://localhost:80" {
		t.Fatalf("dev-профиль потерян при переключении: %+v", got)
	}
}

func TestApplyProfileUnknownEnvTreatedAsProd(t *testing.T) {
	withBuildDefaults(t, "https://prod.example", "prod.example", "25565")
	cfg := &Config{Env: "staging"}

	cfg.ApplyProfile(cfg.EnvName())
	if cfg.Env != envProd || cfg.ApiBaseUrl != "https://prod.example" {
		t.Fatalf("неизвестное окружение не свелось к проду: %+v", cfg)
	}
}
