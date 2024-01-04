NCLET_IMG ?= netcon-pms-nclet:dev

DEFAULT_SSH_USERNAME ?= username
DEFAULT_SSH_PASSWORD ?= password

##@ Build: nclet

.PHONY: nclet-docker-build
nclet-docker-build: test ## Build docker image with the manager.
	DOCKER_BUILDKIT=1 docker build --file cmd/nclet/Dockerfile \
		--build-arg DEFAULT_SSH_USERNAME="${DEFAULT_SSH_USERNAME}" \
		--build-arg DEFAULT_SSH_PASSWORD="${DEFAULT_SSH_PASSWORD}" \
		-t ${NCLET_IMG} .

.PHONY: nclet-docker-push
nclet-docker-push: ## Push docker image with the manager.
	docker push ${NCLET_IMG}
