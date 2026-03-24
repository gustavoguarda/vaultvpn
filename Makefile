BINARY=vaultvpn
MAIN=./cmd/vaultvpn/
GUI=./cmd/vaultvpn-gui/
FYNE=$(shell go env GOPATH)/bin/fyne

.PHONY: build run clean linux app

build:
	go build -o $(BINARY) $(MAIN)

run: build
	./$(BINARY)

app:
	$(FYNE) package --target darwin --name VaultVPN --id com.gustavoguarda.vaultvpn --src $(GUI)

clean:
	rm -f $(BINARY) $(BINARY)-linux-*

linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux-amd64 $(MAIN)
	GOOS=linux GOARCH=arm64 go build -o $(BINARY)-linux-arm64 $(MAIN)
