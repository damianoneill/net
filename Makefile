# REQUIRED SECTION
include ./.golang.mk
# END OF REQUIRED SECTION

# Run 'make help' for the list of default targets

# Example of overriding of default target
#test: ## run test with coverage using the vendor directory
#	go test -mod vendor -v -cover ./... -coverprofile cover.out

build: ## build the v2 code
	@echo ">>> go build v2"
	@cd v2 && $(GO) build -ldflags="$(LD_FLAGS)" ./...

test: ## run test with coverage, v2 only
	@echo ">>> go test v2"
	@cd v2 && $(GO) test -v -cover ./... -coverprofile ../cover.out

lint: ## run golangci-lint v2 using the configuration in .golangci.yml
	@echo ">>> golangci-lint run v2"
	@cd v2 && $(GOBIN)/golangci-lint run -c ../.golangci.yml

# Threshold increased from default
coverage: test
	goverreport -coverprofile=cover.out -sort=block -order=desc -threshold=91

runner:
	@echo ">>> not supported in this project"

licenses:
	@echo ">>> not supported in this project"

scanner:
	@echo ">>> not supported in this project"
