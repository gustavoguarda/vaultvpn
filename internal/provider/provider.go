package provider

import "fmt"

type Credentials struct {
	Username string
	Password string
	TOTP     string
}

type CredentialProvider interface {
	Name() string
	Unlock(masterPassword string) (string, error)
	GetCredentials(session, itemName string) (*Credentials, error)
}

var registry = map[string]CredentialProvider{}

func Register(p CredentialProvider) {
	registry[p.Name()] = p
}

func Get(name string) (CredentialProvider, error) {
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("provider não encontrado: %s", name)
	}
	return p, nil
}

func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
