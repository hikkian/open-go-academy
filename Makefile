GO := sh scripts/go.sh
IMAGE := docker.io/library/golang@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8
.PHONY: bootstrap build dev runner test sandbox-test e2e check
bootstrap:
	$(GO) mod download
	npm ci
	cd web && npm ci
	npx playwright install chromium
	podman pull $(IMAGE)
build:
	mkdir -p .tools
	$(GO) build -o .tools/academy-api ./cmd/api
	$(GO) build -o .tools/academy-runner ./cmd/runner
	cd web && npm run build
dev:
	$(GO) run ./cmd/api
runner:
	$(GO) run ./cmd/runner
test:
	$(GO) test ./...
	$(GO) -C content/projects/p1/reference test ./...
	$(GO) -C content/projects/p1/reference vet ./...
	$(GO) vet ./...
	$(GO) test -race ./...
	cd web && npm run typecheck && npm run build
sandbox-test:
	ACADEMY_SANDBOX_TEST=1 $(GO) test -count=1 -timeout=10m -v ./internal/runner
e2e:
	ACADEMY_E2E_RUNNER=1 npm run e2e
check: test
