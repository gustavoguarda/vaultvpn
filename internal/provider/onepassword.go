package provider

import (
	"fmt"
	"os/exec"
	"strings"
)

type OnePassword struct{}

func init() {
	Register(&OnePassword{})
}

func (o *OnePassword) Name() string { return "1password" }

func (o *OnePassword) Unlock(masterPassword string) (string, error) {
	cmd := exec.Command("op", "signin", "--raw")
	cmd.Stdin = strings.NewReader(masterPassword + "\n")

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("falha ao desbloquear 1Password: %w", err)
	}

	session := strings.TrimSpace(string(out))
	if session == "" {
		return "", fmt.Errorf("sessão do 1Password vazia")
	}
	return session, nil
}

func (o *OnePassword) GetCredentials(session, itemName string) (*Credentials, error) {
	username, err := opGet(session, itemName, "username")
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar username: %w", err)
	}

	password, err := opGet(session, itemName, "password")
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar password: %w", err)
	}

	totp, err := opGetOTP(session, itemName)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar TOTP: %w", err)
	}

	return &Credentials{
		Username: username,
		Password: password,
		TOTP:     totp,
	}, nil
}

func opGet(session, itemName, field string) (string, error) {
	cmd := exec.Command("op", "item", "get", itemName,
		"--fields", "label="+field,
		"--session", session,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "", fmt.Errorf("campo %s vazio", field)
	}
	return val, nil
}

func opGetOTP(session, itemName string) (string, error) {
	cmd := exec.Command("op", "item", "get", itemName,
		"--otp",
		"--session", session,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "", fmt.Errorf("TOTP vazio")
	}
	return val, nil
}
