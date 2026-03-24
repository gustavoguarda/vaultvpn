package vpn

type Status int

const (
	Disconnected Status = iota
	Connecting
	Connected
)

type Runner interface {
	Connect(configPath string, username, fullPassword string, logFn func(string)) error
	Disconnect() error
	Status() Status
}
