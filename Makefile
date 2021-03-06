.PHONY: build docker-build docker-buildx-push help
.DEFAULT_GOAL := help

VERSION := $(shell git describe --always --tags)
BUILD_TIME=$(shell date '+%Y%m%d-%H%M%S')
DOCKER_IMAGE_NAME="spx01/fritzbox-prometheus"
BINARY_NAME=fritzbox-prometheus
BIN_OUT_DIR=bin



build:  ## Build binary
	go build -v -ldflags="-w -s" -o $(BIN_OUT_DIR)/$(BINARY_NAME)


docker-buildx-push:  ## Build multi arch docker images and push
	docker buildx build \
            --platform linux/386,linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64 \
            --tag ${DOCKER_IMAGE_NAME}:${VERSION} --push .
	docker buildx build \
            --platform linux/386,linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64 \
            --tag ${DOCKER_IMAGE_NAME}:latest --push . 

docker-build:  ## Build docker image
	docker build --tag ${DOCKER_IMAGE_NAME} .

help:  ## Shows help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'