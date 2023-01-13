##@ Build: kubectl-netcon

.PHONY: kubectl-netcon-build
kubectl-netcon-build: ## Build kubectl-netcon
	go build -o kubectl-netcon ./cmd/kubectl-netcon
