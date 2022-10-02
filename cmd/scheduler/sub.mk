SCHEDULER_IMG ?= proelbtn/netcon-pms-scheduler:dev

##@ Build: Scheduler

.PHONY: scheduler-docker-build
scheduler-docker-build: test ## Build docker image with the manager.
	docker build --file cmd/scheduler/Dockerfile -t ${SCHEDULER_IMG} .

.PHONY: scheduler-docker-push
scheduler-docker-push: ## Push docker image with the manager.
	docker push ${SCHEDULER_IMG}

.PHONY: docker-buildx
scheduler-docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${SCHEDULER_IMG} -f Dockerfile.cross
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross
