REGISTRY?=ghcr.io/project-ai-services
IMAGE=minio
TAG?=RELEASE.2025-09-07T16-13-09Z

CONTAINER_BUILDER?=podman
build:
	$(CONTAINER_BUILDER) build --build-arg RELEASE=${TAG} -t $(REGISTRY)/$(IMAGE):$(TAG) .
.PHONY: build

push:
	$(CONTAINER_BUILDER) push $(REGISTRY)/$(IMAGE):$(TAG)
.PHONY: push
