# VaultVPN

Cliente VPN cross-platform com interface gráfica e TUI. Busca credenciais e TOTP automaticamente do seu gerenciador de senhas e conecta via OpenVPN.

Suporta **Bitwarden** e **1Password**. Roda em **macOS** e **Linux**.

## Instalação

### Via binário

Baixe o binário da [página de releases](https://github.com/gustavoguarda/vaultvpn/releases) e coloque no seu `$PATH`.

### Compilando do fonte

```bash
git clone https://github.com/gustavoguarda/vaultvpn.git
cd vaultvpn
make build
```

Para gerar o app macOS (`.app`):

```bash
make app
```

## Setup

### macOS

```bash
# OpenVPN
brew install openvpn

# Gerenciador de senhas (escolha um)
brew install bitwarden-cli  # Bitwarden
brew install 1password-cli  # 1Password
```

### Linux (Ubuntu/Debian)

```bash
# OpenVPN
sudo apt install openvpn

# Go (para compilar)
sudo apt install golang-go

# Dependências do Fyne (interface gráfica)
sudo apt install gcc libgl1-mesa-dev xorg-dev

# Bitwarden CLI
sudo snap install bw

# Ou 1Password CLI
# https://developer.1password.com/docs/cli/get-started/#install
```

### Fazer login no gerenciador de senhas

```bash
# Bitwarden
bw login seu-email@exemplo.com

# 1Password
op signin
```

### Configurar `.env` e `config.ovpn`

```bash
cp .env.example .env
cp config.ovpn.example config.ovpn
```

Edite o `.env`:

```
PROVIDER=bitwarden          # ou 1password
ITEM_NAME=Nome do item      # nome do item no vault
OPENVPN_PATH=/opt/homebrew/sbin/openvpn   # macOS
# OPENVPN_PATH=/usr/sbin/openvpn          # Linux
APP_DIR=~/caminho/para/vaultvpn
```

Edite o `config.ovpn` com a configuração fornecida pelo administrador da VPN.

### Configurar sudoers (evita pedir senha do sistema a cada conexão)

```bash
sudo visudo -f /etc/sudoers.d/openvpn
```

Adicione (substitua pelo seu usuário e caminho do openvpn):

```
# macOS
seu-usuário ALL=(ALL) NOPASSWD: /opt/homebrew/sbin/openvpn, /bin/kill

# Linux
seu-usuário ALL=(ALL) NOPASSWD: /usr/sbin/openvpn, /bin/kill
```

## Uso

```bash
# Interface gráfica (janela) — macOS e Linux
vaultvpn gui

# TUI interativa no terminal (padrão)
vaultvpn

# Modo headless (sem interface)
vaultvpn connect
```

No macOS o app também pode ser aberto pelo Spotlight (`Cmd+Space` → "VaultVPN").

## Estrutura

```
cmd/
  vaultvpn/           CLI + TUI
  vaultvpn-gui/       Entry point da GUI (usado pelo .app)
internal/
  config/             Carregamento do .env
  provider/           Providers (Bitwarden, 1Password)
  vpn/                Runner OpenVPN
  gui/                Interface gráfica (Fyne)
  tui/                Interface terminal (bubbletea)
legacy/               App Swift e script shell originais (macOS)
```

## Segurança

- Credenciais salvas em arquivo temporário com `chmod 600`, apagado automaticamente
- `--auth-nocache` impede o OpenVPN de cachear senhas na memória
- Sessão do vault expira automaticamente após inatividade
- `.env` e `config.ovpn` não são versionados (`.gitignore`)

## Legado

A versão original em Swift (app nativo macOS) e o script shell estão em `legacy/`. Funcionam independentemente mas não recebem mais atualizações.
