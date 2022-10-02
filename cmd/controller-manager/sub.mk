CONTROLLER_MANAGER_IMG ?= proelbtn/netcon-pms-controller-manager:dev

##@ Build: Controller Manager

.PHONY: controller-manager-docker-build
controller-manager-docker-build: test ## Build docker image with the manager.
	docker build --file cmd/controller-manager/Dockerfile -t ${CONTROLLER_MANAGER_IMG} .

.PHONY: controller-manager-docker-push
controller-manager-docker-push: ## Push docker image with the manager.
	docker push ${CONTROLLER_MANAGER_IMG}

.PHONY: docker-buildx
controller-manager-docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${CONTROLLER_MANAGER_IMG} -f Dockerfile.cross
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross
