package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gustavoguarda/vaultvpn/internal/config"
	"github.com/gustavoguarda/vaultvpn/internal/provider"
	"github.com/gustavoguarda/vaultvpn/internal/vpn"
)

// Mensagens do bubbletea
type logMsg string
type connectedMsg struct{}
type disconnectedMsg struct{}
type errorMsg struct{ err error }

type state int

const (
	stateInput state = iota
	stateConnecting
	stateConnected
)

type Model struct {
	cfg      *config.Config
	provider provider.CredentialProvider
	runner   *vpn.OpenVPN

	providers    []string
	providerIdx  int

	state    state
	password string
	logs     []string
	maxLogs  int
	err      string
	quitting bool
}

func NewModel(cfg *config.Config, p provider.CredentialProvider, runner *vpn.OpenVPN) Model {
	providers := provider.List()
	idx := 0
	for i, name := range providers {
		if name == p.Name() {
			idx = i
			break
		}
	}
	return Model{
		cfg:         cfg,
		provider:    p,
		runner:      runner,
		providers:   providers,
		providerIdx: idx,
		state:       stateInput,
		maxLogs:     50,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.state == stateConnected {
				m.runner.Disconnect()
			}
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if m.state == stateInput && m.password != "" {
				m.state = stateConnecting
				m.err = ""
				m.logs = nil
				return m, m.connectCmd()
			}
			if m.state == stateConnected {
				m.logs = append(m.logs, "Desconectando...")
				m.runner.Disconnect()
				m.state = stateInput
				m.password = ""
				m.logs = append(m.logs, "VPN desconectada.")
				return m, nil
			}

		case "tab":
			if m.state == stateInput && len(m.providers) > 1 {
				m.providerIdx = (m.providerIdx + 1) % len(m.providers)
				p, err := provider.Get(m.providers[m.providerIdx])
				if err == nil {
					m.provider = p
				}
			}

		case "backspace":
			if m.state == stateInput && len(m.password) > 0 {
				m.password = m.password[:len(m.password)-1]
			}

		default:
			if m.state == stateInput && len(msg.String()) == 1 {
				m.password += msg.String()
			}
		}

	case logMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > m.maxLogs {
			m.logs = m.logs[len(m.logs)-m.maxLogs:]
		}
		return m, nil

	case connectedMsg:
		m.state = stateConnected
		return m, nil

	case disconnectedMsg:
		m.state = stateInput
		m.password = ""
		return m, nil

	case errorMsg:
		m.state = stateInput
		m.password = ""
		m.err = msg.err.Error()
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render("VaultVPN")
	provLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(fmt.Sprintf(" [%s]", m.provider.Name()))
	b.WriteString(title + provLabel + "\n\n")

	// Status
	switch m.state {
	case stateInput:
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("●")
		b.WriteString(dot + " Desconectado\n\n")
	case stateConnecting:
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("●")
		b.WriteString(dot + " Conectando...\n\n")
	case stateConnected:
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("●")
		b.WriteString(dot + " Conectado\n\n")
	}

	// Error
	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		b.WriteString(errStyle.Render("Erro: "+m.err) + "\n\n")
	}

	// Input or action
	switch m.state {
	case stateInput:
		b.WriteString("Master Password: " + strings.Repeat("*", len(m.password)) + "█\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("enter: conectar  tab: trocar provider  esc: sair") + "\n")
	case stateConnecting:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("esc: cancelar") + "\n")
	case stateConnected:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("enter: desconectar  esc: sair") + "\n")
	}

	// Logs
	if len(m.logs) > 0 {
		b.WriteString("\n")
		logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		// Mostra as últimas 15 linhas
		start := 0
		if len(m.logs) > 15 {
			start = len(m.logs) - 15
		}
		for _, line := range m.logs[start:] {
			b.WriteString(logStyle.Render(line) + "\n")
		}
	}

	return b.String()
}

func (m Model) connectCmd() tea.Cmd {
	return func() tea.Msg {
		password := m.password

		session, err := m.provider.Unlock(password)
		if err != nil {
			return errorMsg{err}
		}

		creds, err := m.provider.GetCredentials(session, m.cfg.ItemName)
		if err != nil {
			return errorMsg{err}
		}

		fullPassword := creds.Password + creds.TOTP

		// Conecta em goroutine pra streamar logs via programa
		program := getProgram()
		if program == nil {
			return errorMsg{fmt.Errorf("programa TUI não inicializado")}
		}

		go func() {
			program.Send(logMsg(fmt.Sprintf("TOTP: %s", creds.TOTP)))
			program.Send(logMsg("Conectando via OpenVPN..."))

			err := m.runner.Connect(m.cfg.ConfigPath, creds.Username, fullPassword, func(line string) {
				program.Send(logMsg(line))
			})

			if err != nil {
				program.Send(errorMsg{err})
			}
			program.Send(disconnectedMsg{})
		}()

		return connectedMsg{}
	}
}

var programRef *tea.Program

func SetProgram(p *tea.Program) {
	programRef = p
}

func getProgram() *tea.Program {
	return programRef
}
