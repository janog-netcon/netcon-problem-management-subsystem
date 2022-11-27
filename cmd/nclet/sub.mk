NCLET_IMG ?= netcon-pms-nclet:dev

DEFAULT_USERNAME ?= username
DEFAULT_PASSWORD ?= password

##@ Build: nclet

.PHONY: nclet-docker-build
nclet-docker-build: test ## Build docker image with the manager.
	DOCKER_BUILDKIT=1 docker build --file cmd/nclet/Dockerfile \
		--build-arg DEFAULT_USERNAME="${DEFAULT_USERNAME}" \
		--build-arg DEFAULT_PASSWORD="${DEFAULT_PASSWORD}" \
		-t ${NCLET_IMG} .

.PHONY: nclet-docker-push
nclet-docker-push: ## Push docker image with the manager.
	docker push ${NCLET_IMG}
