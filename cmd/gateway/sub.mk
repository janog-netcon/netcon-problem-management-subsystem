GATEWAY_IMG ?= netcon-pms-gateway:dev

##@ Build: gateway

.PHONY: gateway-docker-build
gateway-docker-build: test ## Build docker image with the manager.
	DOCKER_BUILDKIT=1 docker build --file cmd/gateway/Dockerfile -t ${GATEWAY_IMG} .

.PHONY: gateway-docker-push
gateway-docker-push: ## Push docker image with the manager.
	docker push ${NCLET_IMG}

.PHONY: gateway-kind-push
gateway-kind-push: ## Push docker image to kind cluster
	kind load docker-image ${GATEWAY_IMG}
