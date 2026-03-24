package vpn

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

type OpenVPN struct {
	binaryPath string
	process    *exec.Cmd
	authFile   string
	status     atomic.Int32
	mu         sync.Mutex
}

func NewOpenVPN(binaryPath string) *OpenVPN {
	o := &OpenVPN{binaryPath: binaryPath}
	o.status.Store(int32(Disconnected))
	return o
}

func (o *OpenVPN) Status() Status {
	return Status(o.status.Load())
}

func (o *OpenVPN) Connect(configPath string, username, fullPassword string, logFn func(string)) error {
	authFile, err := writeAuthFile(username, fullPassword)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo de autenticação: %w", err)
	}
	o.authFile = authFile
	o.status.Store(int32(Connecting))

	cmd := exec.Command("sudo", o.binaryPath,
		"--config", configPath,
		"--auth-user-pass", authFile,
		"--auth-nocache",
		"--verb", "1",
	)

	o.mu.Lock()
	o.process = cmd
	o.mu.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		o.cleanup()
		return fmt.Errorf("falha ao criar pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		o.cleanup()
		return fmt.Errorf("falha ao iniciar OpenVPN: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if logFn != nil {
			logFn(line)
		}

		if strings.Contains(line, "Initialization Sequence Completed") {
			o.status.Store(int32(Connected))
			o.removeAuthFile()
			logFn("VPN conectada!")
		}

		if strings.Contains(line, "AUTH_FAILED") {
			o.cleanup()
			return fmt.Errorf("autenticação VPN falhou")
		}
	}

	// Usa variável local cmd, não o.process (que pode ser nil)
	cmd.Wait()
	o.cleanup()
	return nil
}

func (o *OpenVPN) Disconnect() error {
	o.mu.Lock()
	proc := o.process
	o.mu.Unlock()

	if proc != nil && proc.Process != nil {
		proc.Process.Signal(syscall.SIGTERM)
	}

	o.status.Store(int32(Disconnected))
	o.removeAuthFile()
	return nil
}

func (o *OpenVPN) cleanup() {
	o.mu.Lock()
	o.process = nil
	o.mu.Unlock()
	o.status.Store(int32(Disconnected))
	o.removeAuthFile()
}

func (o *OpenVPN) removeAuthFile() {
	if o.authFile != "" {
		os.Remove(o.authFile)
		o.authFile = ""
	}
}

func writeAuthFile(username, fullPassword string) (string, error) {
	f, err := os.CreateTemp("", "vaultvpn-auth-*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := f.Chmod(0600); err != nil {
		os.Remove(f.Name())
		return "", err
	}

	content := username + "\n" + fullPassword + "\n"
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", err
	}

	return f.Name(), nil
}
