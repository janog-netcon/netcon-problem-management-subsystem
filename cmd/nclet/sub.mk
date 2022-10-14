NCLET_IMG ?= netcon-pms-nclet:dev

##@ Build: nclet

.PHONY: nclet-docker-build
nclet-docker-build: test ## Build docker image with the manager.
	DOCKER_BUILDKIT=1 docker build --file cmd/nclet/Dockerfile -t ${NCLET_IMG} .

.PHONY: nclet-docker-push
nclet-docker-push: ## Push docker image with the manager.
	docker push ${NCLET_IMG}
