# REQUIRED SECTION
include ./.golang.mk
# END OF REQUIRED SECTION

# Run 'make help' for the list of default targets

# Example of overriding of default target
#test: ## run test with coverage using the vendor directory
#	go test -mod vendor -v -cover ./... -coverprofile cover.out

# Threshold increased from default
coverage: test
	goverreport -coverprofile=cover.out -sort=block -order=desc -threshold=92

runner:
	@echo ">>> not supported in this project"

licenses:
	@echo ">>> not supported in this project"

scanner:
	@echo ">>> not supported in this project"
