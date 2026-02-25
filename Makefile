SHELL := /bin/bash
.PHONY: build build-windows build-client build-client-all clean dev package package-insecure help
.DEFAULT_GOAL := help

BINARY = winshut
WINDOWS_BINARY = winshut.exe
CLIENT_BINARY = winshut-client
CLIENT_DIR = ./cmd/winshut-client
DIST_DIR = dist

LDFLAGS = -s -w

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build server for current platform
	go build -o $(BINARY) .

build-windows: ## Cross-compile server for Windows amd64
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BINARY) -ldflags="$(LDFLAGS)" .

build-client: ## Build client for current platform
	go build -o $(CLIENT_BINARY) $(CLIENT_DIR)

build-client-all: ## Cross-compile client for all platforms (linux/mac/windows, amd64/arm64)
	@mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/winshut-client-linux-amd64       $(CLIENT_DIR)
	GOOS=linux   GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/winshut-client-linux-arm64       $(CLIENT_DIR)
	GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/winshut-client-darwin-amd64      $(CLIENT_DIR)
	GOOS=darwin  GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/winshut-client-darwin-arm64      $(CLIENT_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/winshut-client-windows-amd64.exe $(CLIENT_DIR)
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/winshut-client-windows-arm64.exe $(CLIENT_DIR)
	@echo "Client binaries written to $(DIST_DIR)/"

clean: ## Remove build artifacts
	rm -f $(BINARY) $(WINDOWS_BINARY) $(CLIENT_BINARY)
	rm -rf $(DIST_DIR)

# Generate certs (CA, server, client) for development/testing
# Prompts for SAN entries (hostnames and IPs for the server cert)
dev-certs: ## Generate dev CA, server, and client certs
	@mkdir -p certs
	@read -p "Enter SANs (comma-separated hostnames/IPs, e.g. mypc.local,192.168.1.100): " SANS; \
	SAN_EXT=""; \
	DNS_IDX=1; \
	IP_IDX=1; \
	IFS=',' read -ra ENTRIES <<< "$$SANS"; \
	for entry in "$${ENTRIES[@]}"; do \
		entry=$$(echo "$$entry" | xargs); \
		if [[ "$$entry" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
			SAN_EXT="$${SAN_EXT}IP.$$IP_IDX = $$entry\n"; \
			IP_IDX=$$((IP_IDX + 1)); \
		else \
			SAN_EXT="$${SAN_EXT}DNS.$$DNS_IDX = $$entry\n"; \
			DNS_IDX=$$((DNS_IDX + 1)); \
		fi; \
	done; \
	echo "Generating CA..."; \
	openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
		-keyout certs/ca.key -out certs/ca.crt -days 3650 -nodes \
		-subj "/CN=WinShut CA" 2>/dev/null; \
	echo "Generating server cert (SANs: $$SANS)..."; \
	openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
		-keyout certs/server.key -out certs/server.csr -nodes \
		-subj "/CN=winshut" 2>/dev/null; \
	printf "[SAN]\nsubjectAltName=@alt_names\n[alt_names]\n$$SAN_EXT" > certs/san.cnf; \
	openssl x509 -req -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key \
		-CAcreateserial -out certs/server.crt -days 365 \
		-extensions SAN -extfile certs/san.cnf 2>/dev/null; \
	echo "Generating client cert..."; \
	openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
		-keyout certs/client.key -out certs/client.csr -nodes \
		-subj "/CN=winshut-client" 2>/dev/null; \
	openssl x509 -req -in certs/client.csr -CA certs/ca.crt -CAkey certs/ca.key \
		-CAcreateserial -out certs/client.crt -days 365 2>/dev/null; \
	rm -f certs/server.csr certs/client.csr certs/ca.srl certs/san.cnf; \
	echo "Certs written to certs/ (ca, server, client)"

package: build-windows build-client-all dev-certs ## Package with public certs only
	rm -f winshut.zip
	zip winshut.zip $(WINDOWS_BINARY) $(DIST_DIR)/winshut-client-* certs/ca.crt certs/server.crt certs/client.crt
	@echo "Created winshut.zip (public certs only)"

package-insecure: build-windows build-client-all dev-certs ## Package with all certs including private keys
	rm -f winshut.zip
	zip winshut.zip $(WINDOWS_BINARY) $(DIST_DIR)/winshut-client-* certs/ca.crt certs/ca.key certs/server.crt certs/server.key certs/client.crt certs/client.key
	@echo "Created winshut.zip (includes private keys!)"

dev: build dev-certs ## Run server locally with dry-run mode
	./$(BINARY) --cert certs/server.crt --key certs/server.key --ca certs/ca.crt --dry-run
