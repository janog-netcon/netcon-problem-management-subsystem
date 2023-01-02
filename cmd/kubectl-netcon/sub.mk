##@ Build: kubectl-netcon

.PHONY: kubectl-netcon-docker-build
kubectl-netcon-docker-build: ## Build kubectl-netcon
	go build -o kubectl-netcon ./cmd/kubectl-netcon
