package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Provider    string
	ItemName    string
	OpenVPNPath string
	AppDir      string
	ConfigPath  string
}

func Load(dir string) (*Config, error) {
	envPath := filepath.Join(dir, ".env")
	env, err := parseEnv(envPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler .env: %w", err)
	}

	// ITEM_NAME é o campo genérico, BW_ITEM_NAME mantido por compatibilidade
	itemName := env["ITEM_NAME"]
	if itemName == "" {
		itemName = env["BW_ITEM_NAME"]
	}

	cfg := &Config{
		Provider:    getOrDefault(env, "PROVIDER", "bitwarden"),
		ItemName:    itemName,
		OpenVPNPath: getOrDefault(env, "OPENVPN_PATH", defaultOpenVPNPath()),
		AppDir:      getOrDefault(env, "APP_DIR", dir),
	}

	cfg.ConfigPath = filepath.Join(dir, "config.ovpn")

	if cfg.ItemName == "" {
		return nil, fmt.Errorf("ITEM_NAME não definido no .env")
	}

	return cfg, nil
}

func parseEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return env, scanner.Err()
}

func getOrDefault(env map[string]string, key, fallback string) string {
	if v, ok := env[key]; ok && v != "" {
		return v
	}
	return fallback
}

func defaultOpenVPNPath() string {
	switch os := strings.ToLower(os.Getenv("GOOS")); {
	case os == "linux":
		return "/usr/sbin/openvpn"
	default:
		return "/opt/homebrew/sbin/openvpn"
	}
}
