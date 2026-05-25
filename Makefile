# vmux build orchestration. See README.md for prerequisites.
.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

REPO_ROOT := $(shell pwd)
DERIVED_DATA := $(REPO_ROOT)/build/DerivedData

.PHONY: help deps sidecar build dev dmg test clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

deps: ## Generate the Xcode project from project.yml (XcodeGen)
	cd app && xcodegen generate

sidecar: ## Build the universal (arm64+amd64) Go sidecar binary
	./scripts/build-sidecar.sh

build: ## Build the universal sidecar + the app (Debug, unsigned)
	./scripts/build-app.sh

dev: ## Build everything and launch the app locally
	./scripts/dev.sh

dmg: ## Build a signed + notarized DMG (needs Developer ID env vars)
	./scripts/make-dmg.sh

test: ## Run Go sidecar tests + Swift unit tests
	cd sidecar && go vet ./... && go test ./...
	@if [ -d app/Vmux.xcodeproj ] || command -v xcodegen >/dev/null; then \
		cd app && xcodegen generate >/dev/null && \
		xcodebuild -project Vmux.xcodeproj -scheme Vmux \
			-configuration Debug -derivedDataPath "$(DERIVED_DATA)" \
			CODE_SIGNING_ALLOWED=NO build 1>/dev/null && \
		echo "Swift build OK"; \
	fi

clean: ## Remove build artifacts
	rm -rf build dist sidecar/bin app/Vmux.xcodeproj
	cd sidecar && go clean -cache 2>/dev/null || true
