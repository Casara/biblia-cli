# Dev tools, version-pinned via `go run` instead of a globally installed
# binary, so contributors and CI run exactly the same versions.
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GO_ARCH_LINT  := github.com/fe3dback/go-arch-lint@v1.18.0
# Unpinned on purpose: govulncheck is only useful when it knows about the
# vulnerabilities disclosed since the last release.
GOVULNCHECK   := golang.org/x/vuln/cmd/govulncheck@latest

MARKDOWNLINT := markdownlint-cli2@0.23.2
VSCE         := @vscode/vsce@4.1.0

PREFIX ?= $(HOME)/.local
# Only the CLI's tags (v0.1.0); the editor plugins have their own
# (vscode-v0.2.0, sublime-0.2.0).
VERSION ?= $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo dev)

# Sublime Text 4 keeps using ~/.config/sublime-text-3 when it was upgraded
# from version 3, and ~/.config/sublime-text on a fresh install: take the
# one that exists. Override with SUBLIME_PACKAGES=... if neither fits.
SUBLIME_PACKAGES ?= $(firstword $(wildcard $(HOME)/.config/sublime-text-3/Packages \
	$(HOME)/.config/sublime-text/Packages))

.PHONY: help build install run fmt lint lint-fix lint-md arch-lint vuln \
	test test-race coverage check clean \
	sublime sublime-release vscode

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## Build the CLI (bin/biblia)
	@go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/biblia ./cmd/biblia

install: build ## Install the CLI to $(PREFIX)/bin (default ~/.local/bin)
	@install -Dm755 bin/biblia $(PREFIX)/bin/biblia
	@echo "instalado em $(PREFIX)/bin/biblia"

run: ## Run the CLI (pass arguments via ARGS, e.g. make run ARGS="jo 3.16 -v ara,nvi")
	@go run ./cmd/biblia $(ARGS)

fmt: ## Format the code (gofmt/goimports, via golangci-lint)
	@go run $(GOLANGCI_LINT) fmt

lint: ## Run the linter (golangci-lint v2, default:all - see .golangci.yml)
	@go run $(GOLANGCI_LINT) run

lint-fix: ## Run the linter and apply available auto-fixes
	@go run $(GOLANGCI_LINT) run --fix

lint-md: ## Lint markdown files (markdownlint-cli2, requires Node.js >= 20)
	@npx --yes $(MARKDOWNLINT) "**/*.md" "#NOTES.md" "#TEMP.md" "#editors/vscode/node_modules"

arch-lint: ## Check the package dependency graph (.go-arch-lint.yml)
	@go run $(GO_ARCH_LINT) check

vuln: ## Scan for known vulnerabilities reachable from this code (govulncheck)
	@go run $(GOVULNCHECK) ./...

test: ## Run the tests (no network: the server is faked)
	@go test ./...

test-race: ## Run the tests with the race detector
	@go test -race ./...

coverage: ## Generate coverage.out and coverage.html
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Report at coverage.html"

check: lint arch-lint test-race ## Run lint + arch-lint + tests with the race detector (what CI runs)

sublime: ## Link the Sublime Text plugin into the Packages folder (sublime-text-3 or sublime-text)
	@test -n "$(SUBLIME_PACKAGES)" || { echo "pasta Packages do Sublime não encontrada em ~/.config/sublime-text-3 nem em ~/.config/sublime-text (use SUBLIME_PACKAGES=...)"; exit 1; }
	@ln -sfn "$(CURDIR)/editors/sublime" "$(SUBLIME_PACKAGES)/Biblia"
	@echo "ligado: $(SUBLIME_PACKAGES)/Biblia -> editors/sublime"

vscode: ## Package the VS Code extension and install it (requires Node.js and `code`)
	@mkdir -p bin
	@cd editors/vscode && npx --yes $(VSCE) package --no-dependencies --skip-license -o ../../bin/biblia.vsix
	@code --install-extension bin/biblia.vsix --force

# The Sublime package is published from its own repository, as Package
# Control requires (the package at the repository root); this tree stays
# the source. "make sublime-release V=0.2.0" pushes editors/sublime there
# as the main branch, tags it 0.2.0 (Package Control reads the tags) and
# tags this repository sublime-0.2.0.
SUBLIME_MIRROR ?= git@github.com:Casara/biblia-sublime.git

sublime-release: ## Publish editors/sublime to the Package Control mirror: make sublime-release V=0.2.0
	@test -n "$(V)" || { echo "use: make sublime-release V=0.2.0"; exit 1; }
	@git diff --quiet HEAD -- editors/sublime && git diff --cached --quiet -- editors/sublime \
		|| { echo "editors/sublime tem mudanças sem commit"; exit 1; }
	@git subtree split --prefix=editors/sublime -b sublime-dist
	@git push $(SUBLIME_MIRROR) sublime-dist:refs/heads/main
	@git push $(SUBLIME_MIRROR) sublime-dist:refs/tags/$(V)
	@git tag -a sublime-$(V) -m "Sublime package $(V)" HEAD
	@git push origin sublime-$(V)
	@echo "publicado: $(SUBLIME_MIRROR) main e tag $(V); tag sublime-$(V) aqui"

clean: ## Remove build/test artifacts
	@rm -rf bin coverage.out coverage.html mutation.json
