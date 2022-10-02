NCLET_IMG ?= proelbtn/netcon-pms-nclet:dev

##@ Build: nclet

.PHONY: nclet-docker-build
nclet-docker-build: test ## Build docker image with the manager.
	docker build --file cmd/nclet/Dockerfile -t ${NCLET_IMG} .

.PHONY: nclet-docker-push
nclet-docker-push: ## Push docker image with the manager.
	docker push ${NCLET_IMG}

.PHONY: nclet-docker-buildx
nclet-docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${NCLET_IMG} -f Dockerfile.cross
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross
