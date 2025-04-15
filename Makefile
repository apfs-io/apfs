include .env
export

IMAGE_NAME=github.com/apfs-io/apfs
DOCKER_CONTAINER_IMAGE=${IMAGE_NAME}:latest
DOCKER_CONTAINER_TESTAPP_IMAGE=${IMAGE_NAME}-testapp:latest

APP_TAGS=${BUILD_TAGS}

DCMD:=docker compose -p apfs -f deploy/develop/docker-compose.yml

include deploy/build.mk

.PHONY: all
all: lint cover

.PHONY: lint
lint: golint

.PHONY: golint
golint:
	# golint -set_exit_status ./...
	golangci-lint run -v ./...

.PHONY: fmt
fmt: ## Run formatting code
	@echo "Fix formatting"
	@gofmt -w ${GO_FMT_FLAGS} $$(go list -f "{{ .Dir }}" ./...); if [ "$${errors}" != "" ]; then echo "$${errors}"; fi

.PHONY: generate-code
generate-code: ## Run codegeneration procedure
	@go generate ./...

# .PHONY: license
# license: __eval_srcs $(UPDATE_LICENSE)
# 	update-license --owner="TrafficStars LTD" $(SRCS)

.PHONY: build_proto
build_proto: ## Build protocol buffers
	cd protocol && buf generate

build_proto_update:
	cd protocol && buf dep update

.PHONY: test
test: ## Run tests
	go test -race -v ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: vendor
vendor: ## Run go mod vendor
	go mod vendor

.PHONY: cover
cover:
	@mkdir -p $(TMP_ETC)
	@rm -f $(TMP_ETC)/coverage.txt $(TMP_ETC)/coverage.html
	go test -race -coverprofile=$(TMP_ETC)/coverage.txt -coverpkg=./... ./...
	@go tool cover -html=$(TMP_ETC)/coverage.txt -o $(TMP_ETC)/coverage.html
	@echo
	@go tool cover -func=$(TMP_ETC)/coverage.txt | grep total
	@echo
	@echo Open the coverage report:
	@echo open $(TMP_ETC)/coverage.html

.PHONY: __eval_srcs
__eval_srcs:
	$(eval SRCS := $(shell find . -not -path 'bazel-*' -not -path '.tmp*' -name '*.go'))

.PHONY: build
build: ## Build application
	@echo "Build application"
	@$(call do_build,"cmd/apfs/main.go",apfs)
	@cp "${shell go env GOROOT}/lib/time/zoneinfo.zip" .build/zoneinfo.zip
	@mkdir -p .build/.empty
	@chmod 777 .build/.empty

.PHONY: build-testapp
build-testapp: ## Build test application
	@echo "Build test application"
	@$(call do_build,"cmd/testapp/main.go",testapp)

.PHONY: build-docker-dev
build-docker-dev: build
	echo "Build develop docker image"
	DOCKER_BUILDKIT=${DOCKER_BUILDKIT} docker build -t ${DOCKER_CONTAINER_IMAGE} -f deploy/develop/apfs.dockerfile .

.PHONY: build-docker-testapp
build-docker-testapp: build-testapp
	echo "Build test app docker image"
	DOCKER_BUILDKIT=${DOCKER_BUILDKIT} docker build -t ${DOCKER_CONTAINER_TESTAPP_IMAGE} -f deploy/develop/testapp.dockerfile .

.PHONY: buildx-docker-production
buildx-docker-production:
	echo "Build production docker image"
	docker buildx build \
		--platform linux/amd64,linux/arm/v7,linux/arm64/v8 \
		-t ${IMAGE_NAME}:ubunty-imagemagick -f deploy/production/ubuntu-imagemagick.dockerfile .
	docker buildx build \
		--platform linux/amd64,linux/arm/v7,linux/arm64/v8 \
		-t ${IMAGE_NAME}:ubuntu -f deploy/production/ubuntu.dockerfile .
	docker buildx build \
		--platform linux/amd64,linux/arm/v7,linux/arm64/v8 \
		-t ${IMAGE_NAME}:debian -f deploy/production/debian.dockerfile .
	docker buildx build \
		--platform ${DOCKER_PLATFORM_LIST} \
		-t ${IMAGE_NAME}:scratch -f deploy/production/scratch.dockerfile .

.PHONY: clean
clean: ## Clean build files
	@rm -rf .build

.PHONY: run
run: devdocker_build_test ## Run application
	${DCMD} run --rm --service-ports server

.PHONY: runtest
runtest: devdocker_build_test ## Run test application
	${DCMD} run --rm --service-ports test

.PHONY: stop
stop: ## Stop all services
	${DCMD} stop

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
