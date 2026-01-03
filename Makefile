.PHONY: build install test clean proto

VERSION ?= 0.0.1
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

LDFLAGS := -X github.com/synheart/synheart-cli/internal/cli.Version=$(VERSION) -X github.com/synheart/synheart-cli/internal/cli.Commit=$(COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/synheart cmd/synheart/main.go

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/synheart
	@# Optional: install shell completion scripts.
	@# - Non-interactive: set INSTALL_COMPLETIONS to one of: zsh|bash|fish|powershell
	@#   Example: make install INSTALL_COMPLETIONS=zsh
	@# - Interactive: if INSTALL_COMPLETIONS is empty and stdin is a TTY, we'll ask.
	@set -e; \
	BIN_DIR="$$(go env GOBIN)"; \
	if [ -z "$$BIN_DIR" ]; then BIN_DIR="$$(go env GOPATH)/bin"; fi; \
	BIN="$$BIN_DIR/synheart"; \
	SHELL_NAME="$(INSTALL_COMPLETIONS)"; \
	if [ -z "$$SHELL_NAME" ] && [ -t 0 ]; then \
		printf "\nInstall shell completion? [zsh/bash/fish/powershell/skip] (default: skip): " ; \
		read -r SHELL_NAME || true ; \
	fi; \
	case "$$SHELL_NAME" in \
		""|"skip"|"no"|"n") \
			printf "\nSkipping completion install. You can run: synheart completion zsh|bash|fish|powershell\n\n" ;; \
		"zsh") \
			mkdir -p "$$HOME/.zsh/completions" ; \
			"$$BIN" completion zsh > "$$HOME/.zsh/completions/_synheart" ; \
			printf "\nInstalled zsh completion to $$HOME/.zsh/completions/_synheart\n" ; \
			printf "Enable it by adding to ~/.zshrc:\n" ; \
			printf "  fpath=(~/.zsh/completions $$fpath)\n  autoload -Uz compinit\n  compinit\n\n" ;; \
		"bash") \
			mkdir -p "$$HOME/.bash_completion.d" ; \
			"$$BIN" completion bash > "$$HOME/.bash_completion.d/synheart" ; \
			printf "\nInstalled bash completion to $$HOME/.bash_completion.d/synheart\n" ; \
			printf "Enable it by sourcing it (e.g. in ~/.bashrc):\n" ; \
			printf "  source ~/.bash_completion.d/synheart\n\n" ;; \
		"fish") \
			mkdir -p "$$HOME/.config/fish/completions" ; \
			"$$BIN" completion fish > "$$HOME/.config/fish/completions/synheart.fish" ; \
			printf "\nInstalled fish completion to $$HOME/.config/fish/completions/synheart.fish\n\n" ;; \
		"powershell") \
			printf "\nPowerShell completion script:\n\n" ; \
			"$$BIN" completion powershell ; \
			printf "\n\n" ;; \
		*) \
			printf "\nUnknown INSTALL_COMPLETIONS=%s (expected zsh|bash|fish|powershell|skip)\n\n" "$$SHELL_NAME" ; \
			exit 2 ;; \
	esac

test:
	go test ./...

test-race:
	go test -race ./...

clean:
	rm -rf bin/

proto:
	mkdir -p internal/proto/hsi
	protoc --go_out=internal/proto/hsi --go_opt=paths=source_relative proto/hsi.proto
