.PHONY: dev build build-rpi clean install-deps

# Host application (Wails)
dev:
	wails dev

build:
	wails build

# RPi daemon cross-compilation (embeds build timestamp as version)
build-rpi:
	cd rpi-daemon && GOOS=linux GOARCH=arm64 go build \
		-ldflags "-X main.buildVersion=$$(date +%Y%m%d-%H%M%S)" \
		-o ../build/bin/kvm-daemon .

# Deploy RPi daemon to target
deploy-rpi: build-rpi
	@echo "Usage: scp build/bin/kvm-daemon pi@<rpi-ip>:/usr/local/bin/"
	@echo "       scp rpi-daemon/kvm-daemon.service pi@<rpi-ip>:/etc/systemd/system/"

# Install frontend dependencies
install-deps:
	cd frontend && npm install

# Clean build artifacts
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
