package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Bitwarden struct{}

func init() {
	Register(&Bitwarden{})
}

func (b *Bitwarden) Name() string { return "bitwarden" }

func (b *Bitwarden) Unlock(masterPassword string) (string, error) {
	// Garante PATH com Homebrew antes de qualquer coisa
	ensurePath()

	// Verifica se bw está no PATH
	if _, err := exec.LookPath("bw"); err != nil {
		return "", fmt.Errorf("Bitwarden CLI (bw) não encontrado. Instale com: brew install bitwarden-cli")
	}

	// Checa status do vault
	status, err := bwStatus()
	if err != nil {
		return "", fmt.Errorf("falha ao checar status do Bitwarden: %w", err)
	}

	switch status {
	case "unauthenticated":
		return "", fmt.Errorf("Bitwarden não autenticado. Execute 'bw login' no terminal primeiro")
	case "locked":
		return bwUnlock(masterPassword)
	case "unlocked":
		// Tenta desbloquear mesmo assim pra obter session token
		return bwUnlock(masterPassword)
	default:
		return "", fmt.Errorf("status desconhecido do Bitwarden: %s", status)
	}
}

func (b *Bitwarden) GetCredentials(session, itemName string) (*Credentials, error) {
	username, err := bwGet("username", itemName, session)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar username: %w", err)
	}

	password, err := bwGet("password", itemName, session)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar password: %w", err)
	}

	totp, err := bwGet("totp", itemName, session)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar TOTP: %w", err)
	}

	return &Credentials{
		Username: username,
		Password: password,
		TOTP:     totp,
	}, nil
}

func bwStatus() (string, error) {
	cmd := bwCmd("bw", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", fmt.Errorf("resposta inválida do bw status: %s", string(out))
	}
	return result.Status, nil
}

func bwUnlock(masterPassword string) (string, error) {
	cmd := bwCmd("bw", "unlock", "--passwordenv", "BW_MASTER_PASS", "--raw")
	cmd.Env = append(cmd.Environ(), "BW_MASTER_PASS="+masterPassword)

	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if isCorruptedVault(detail) {
			bwLogout()
			return "", fmt.Errorf("Vault corrompido detectado. Foi feito logout automático.\nExecute 'bw login' no terminal e tente novamente")
		}
		if detail != "" {
			return "", fmt.Errorf("%s", detail)
		}
		return "", fmt.Errorf("falha ao desbloquear: %w", err)
	}

	session := strings.TrimSpace(string(out))
	if session == "" {
		return "", fmt.Errorf("sessão do Bitwarden vazia")
	}
	return session, nil
}

func isCorruptedVault(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "model state is invalid") ||
		strings.Contains(lower, "encrypted migrator")
}

func bwLogout() {
	cmd := bwCmd("bw", "logout")
	_ = cmd.Run()
}

func bwGet(field, itemName, session string) (string, error) {
	cmd := bwCmd("bw", "get", field, itemName, "--session", session)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return "", fmt.Errorf("%s", detail)
		}
		return "", err
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "", fmt.Errorf("campo %s vazio", field)
	}
	return val, nil
}

// ensurePath adiciona Homebrew ao PATH do processo (necessário quando aberto via Spotlight/Finder)
func ensurePath() {
	if runtime.GOOS != "darwin" {
		return
	}
	path := os.Getenv("PATH")
	changed := false
	for _, p := range []string{"/opt/homebrew/bin", "/opt/homebrew/sbin", "/usr/local/bin"} {
		if !strings.Contains(path, p) {
			path = p + ":" + path
			changed = true
		}
	}
	if changed {
		os.Setenv("PATH", path)
	}
}

// bwCmd cria um exec.Command com o ambiente atual (PATH já ajustado por ensurePath)
func bwCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
