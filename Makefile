.PHONY: lint test coverage

GOLANGCI_LINT ?= golangci-lint
BUSINESS_PACKAGES := ./internal/domain ./internal/service ./internal/price/...
BUSINESS_COVERPKG := smartbid-backend/internal/domain,smartbid-backend/internal/service,smartbid-backend/internal/price/...
COVERAGE_MIN ?= 80.0

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
	go test $(BUSINESS_PACKAGES) -coverpkg=$(BUSINESS_COVERPKG) -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out | tail -n 1
	@coverage=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	awk -v coverage="$$coverage" -v minimum="$(COVERAGE_MIN)" 'BEGIN { \
		if (coverage + 0 < minimum + 0) { \
			printf "Business coverage %.1f%% is below required %.1f%%\n", coverage, minimum; \
			exit 1; \
		} \
	}'
