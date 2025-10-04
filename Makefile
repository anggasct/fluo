.PHONY: build test clean example lint fmt vet

# Build the library
build:
	go build ./...

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run the example
example:
	cd examples/traffic-light && go run main.go

# Clean build artifacts
clean:
	go clean ./...
	rm -f coverage.out coverage.html

# Lint the code
lint:
	golangci-lint run

# Format the code
fmt:
	go fmt ./...

# Vet the code
vet:
	go vet ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run all checks
check: fmt vet test

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the library"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  example       - Run the example"
	@echo "  clean         - Clean build artifacts"
	@echo "  lint          - Lint the code"
	@echo "  fmt           - Format the code"
	@echo "  vet           - Vet the code"
	@echo "  deps          - Install dependencies"
	@echo "  check         - Run all checks (fmt, vet, test)"
	@echo "  help          - Show this help"
