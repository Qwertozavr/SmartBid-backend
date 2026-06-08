.PHONY: lint test coverage

GOLANGCI_LINT ?= golangci-lint

lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { \
		echo "golangci-lint is not installed. Install it with:"; \
		echo "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"; \
		exit 1; \
	}
	$(GOLANGCI_LINT) run ./...

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -n 1
