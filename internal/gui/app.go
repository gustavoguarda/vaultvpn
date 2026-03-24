package gui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gustavoguarda/vaultvpn/internal/config"
	"github.com/gustavoguarda/vaultvpn/internal/provider"
	"github.com/gustavoguarda/vaultvpn/internal/vpn"
)

var (
	colorRed    = color.NRGBA{R: 255, G: 80, B: 80, A: 255}
	colorGreen  = color.NRGBA{R: 80, G: 220, B: 100, A: 255}
	colorYellow = color.NRGBA{R: 255, G: 200, B: 50, A: 255}
)

type GUI struct {
	cfg      *config.Config
	provider provider.CredentialProvider
	runner   *vpn.OpenVPN

	app    fyne.App
	window fyne.Window

	statusDot      *canvas.Circle
	statusText     *widget.Label
	providerSelect *widget.Select
	passEntry      *widget.Entry
	actionBtn      *widget.Button
	logText        *widget.Label
	logScroll      *container.Scroll

	connected bool
	logs      string
}

func Run(cfg *config.Config, p provider.CredentialProvider, runner *vpn.OpenVPN) {
	g := &GUI{
		cfg:      cfg,
		provider: p,
		runner:   runner,
	}
	g.build()
	g.window.ShowAndRun()
}

func (g *GUI) build() {
	g.app = app.NewWithID("com.gustavoguarda.vaultvpn")
	g.app.Settings().SetTheme(theme.DarkTheme())

	g.window = g.app.NewWindow("VaultVPN")
	g.window.Resize(fyne.NewSize(420, 350))
	g.window.SetFixedSize(true)
	g.window.CenterOnScreen()

	// Status
	g.statusDot = canvas.NewCircle(colorRed)
	g.statusDot.Resize(fyne.NewSize(12, 12))

	g.statusText = widget.NewLabel("Desconectado")
	g.statusText.TextStyle = fyne.TextStyle{Bold: true}

	// Provider selector
	providers := provider.List()
	g.providerSelect = widget.NewSelect(providers, func(selected string) {
		p, err := provider.Get(selected)
		if err == nil {
			g.provider = p
		}
	})
	g.providerSelect.SetSelected(g.provider.Name())

	statusRow := container.NewHBox(
		container.NewPadded(g.statusDot),
		g.statusText,
		layout.NewSpacer(),
		g.providerSelect,
	)

	// Password
	passLabel := widget.NewLabel("Master Password:")

	g.passEntry = widget.NewPasswordEntry()
	g.passEntry.PlaceHolder = "Digite sua senha..."
	g.passEntry.OnSubmitted = func(_ string) {
		g.onAction()
	}

	// Button
	g.actionBtn = widget.NewButton("Conectar", g.onAction)
	g.actionBtn.Importance = widget.HighImportance

	// Log
	g.logText = widget.NewLabel("")
	g.logText.Wrapping = fyne.TextWrapWord
	g.logText.TextStyle = fyne.TextStyle{Monospace: true}

	g.logScroll = container.NewVScroll(g.logText)
	g.logScroll.SetMinSize(fyne.NewSize(380, 140))

	content := container.NewVBox(
		statusRow,
		widget.NewSeparator(),
		passLabel,
		g.passEntry,
		g.actionBtn,
		widget.NewSeparator(),
		g.logScroll,
	)

	g.window.SetContent(container.NewPadded(content))

	g.window.SetOnClosed(func() {
		g.runner.Disconnect()
	})

	g.window.SetCloseIntercept(func() {
		g.runner.Disconnect()
		g.app.Quit()
	})
}

// appendLog adiciona uma linha ao log — seguro para chamar de qualquer goroutine
func (g *GUI) appendLog(msg string) {
	fyne.Do(func() {
		if g.logs != "" {
			g.logs += "\n"
		}
		g.logs += msg

		// Mantém só as últimas 30 linhas
		lines := 0
		for i := len(g.logs) - 1; i >= 0; i-- {
			if g.logs[i] == '\n' {
				lines++
				if lines >= 30 {
					g.logs = g.logs[i+1:]
					break
				}
			}
		}

		g.logText.SetText(g.logs)
		g.logScroll.ScrollToBottom()
	})
}

func (g *GUI) setStatus(text string, c color.Color) {
	fyne.Do(func() {
		g.statusDot.FillColor = c
		g.statusDot.Refresh()
		g.statusText.SetText(text)
	})
}

func (g *GUI) setConnecting() {
	fyne.Do(func() {
		g.statusDot.FillColor = colorYellow
		g.statusDot.Refresh()
		g.statusText.SetText("Conectando...")
		g.providerSelect.Disable()
		g.passEntry.Disable()
		g.actionBtn.SetText("Conectando...")
		g.actionBtn.Disable()
	})
}

func (g *GUI) setConnected() {
	fyne.Do(func() {
		g.connected = true
		g.statusDot.FillColor = colorGreen
		g.statusDot.Refresh()
		g.statusText.SetText("Conectado")
		g.actionBtn.SetText("Desconectar")
		g.actionBtn.Enable()
	})
}

func (g *GUI) setDisconnected() {
	fyne.Do(func() {
		g.connected = false
		g.statusDot.FillColor = colorRed
		g.statusDot.Refresh()
		g.statusText.SetText("Desconectado")
		g.providerSelect.Enable()
		g.passEntry.Enable()
		g.passEntry.SetText("")
		g.actionBtn.SetText("Conectar")
		g.actionBtn.Enable()
	})
}

func (g *GUI) onAction() {
	if g.connected {
		g.disconnect()
		return
	}
	g.connect()
}

func (g *GUI) connect() {
	password := g.passEntry.Text
	if password == "" {
		g.appendLog("Digite a master password.")
		return
	}

	g.setConnecting()

	go func() {
		g.appendLog(fmt.Sprintf("Desbloqueando %s...", g.provider.Name()))

		session, err := g.provider.Unlock(password)
		if err != nil {
			g.appendLog(fmt.Sprintf("Erro: %v", err))
			g.setDisconnected()
			return
		}
		g.appendLog("Desbloqueado.")

		g.appendLog("Buscando credenciais...")
		creds, err := g.provider.GetCredentials(session, g.cfg.ItemName)
		if err != nil {
			g.appendLog(fmt.Sprintf("Erro: %v", err))
			g.setDisconnected()
			return
		}
		g.appendLog(fmt.Sprintf("TOTP: %s", creds.TOTP))

		fullPassword := creds.Password + creds.TOTP

		g.appendLog("Conectando via OpenVPN...")

		err = g.runner.Connect(g.cfg.ConfigPath, creds.Username, fullPassword, func(line string) {
			g.appendLog(line)
			if g.runner.Status() == vpn.Connected {
				g.setConnected()
			}
		})

		if err != nil {
			g.appendLog(fmt.Sprintf("Erro: %v", err))
		}
		g.setDisconnected()
		g.appendLog("VPN desconectada.")
	}()
}

func (g *GUI) disconnect() {
	g.appendLog("Desconectando...")
	go func() {
		g.runner.Disconnect()
		g.setDisconnected()
		g.appendLog("VPN desconectada.")
	}()
}
