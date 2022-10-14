CONTROLLER_MANAGER_IMG ?= netcon-pms-controller-manager:dev

##@ Build: Controller Manager

.PHONY: controller-manager-docker-build
controller-manager-docker-build: test ## Build docker image with the manager.
	DOCKER_BUILDKIT=1 docker build --file cmd/controller-manager/Dockerfile -t ${CONTROLLER_MANAGER_IMG} .

.PHONY: controller-manager-docker-push
controller-manager-docker-push: ## Push docker image with the manager.
	docker push ${CONTROLLER_MANAGER_IMG}

.PHONY: controller-manager-kind-push
controller-manager-kind-push: ## Push docker image to kind cluster
	kind load docker-image ${CONTROLLER_MANAGER_IMG}
