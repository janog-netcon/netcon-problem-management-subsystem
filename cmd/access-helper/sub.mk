##@ Build: access-helper

.PHONY: access-helper-build
access-helper-build: ## Build access-helper.
	go build -o access-helper ./cmd/access-helper
