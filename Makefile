# Simple Makefile for a Go project

# Build the application
all: build test

fmt:
	@echo "Formatting Go files..."
	@gofmt -w $$(git ls-files '*.go')

gen: templ-install
	@echo "Generating templates..."
	@templ generate
templ-install:
	@if ! command -v templ > /dev/null; then \
		read -p "Go's 'templ' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
		if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
			go install github.com/a-h/templ/cmd/templ@latest; \
			if [ ! -x "$$(command -v templ)" ]; then \
				echo "templ installation failed. Exiting..."; \
				exit 1; \
			fi; \
		else \
			echo "You chose not to install templ. Exiting..."; \
			exit 1; \
		fi; \
	fi
tailwind-install:
	
	@if [ ! -f tailwindcss ]; then curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-x64 -o tailwindcss; fi
	@chmod +x tailwindcss

build: tailwind-install templ-install
	@echo "Building..."
	@$(MAKE) gen
	@./tailwindcss -i internal/web/styles/input.css -o internal/web/assets/css/output.css
	@go build -o main cmd/api/main.go

# Run the application
run: tailwind-install templ-install
	@$(MAKE) gen
	@./tailwindcss -i internal/web/styles/input.css -o internal/web/assets/css/output.css
	@go run cmd/api/main.go

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch: tailwind-install templ-install
	@$(MAKE) gen
	@./tailwindcss -i internal/web/styles/input.css -o internal/web/assets/css/output.css
	@AIR_CMD=""; \
	if command -v air > /dev/null; then \
		AIR_CMD="air"; \
	else \
		read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
		if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
			go install github.com/air-verse/air@latest; \
			AIR_CMD="air"; \
		else \
			echo "You chose not to install air. Exiting..."; \
			exit 1; \
		fi; \
	fi; \
	./tailwindcss -i internal/web/styles/input.css -o internal/web/assets/css/output.css --watch & \
	TW_PID=$$!; \
	trap 'kill $$TW_PID >/dev/null 2>&1' EXIT INT TERM; \
	echo "Watching app and styles..."; \
	$$AIR_CMD

package-macos-app-arm64:
	@./scripts/package_macos_app.sh arm64

package-macos-app-amd64:
	@./scripts/package_macos_app.sh amd64

.PHONY: all fmt gen build run test clean watch tailwind-install templ-install package-macos-app-arm64 package-macos-app-amd64
