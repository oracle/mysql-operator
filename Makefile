# Copyright 2018 Oracle and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ifdef WERCKER
    # Insert swear words about mysql group replication and hostname length. Arghh..
    VERSION ?= ${WERCKER_GIT_COMMIT}
    TENANT := "oracle"
else
    NEW_NAMESPACE ?= e2e-${USER}
    VERSION := ${USER}-$(shell date +%Y%m%d%H%M%S)
    TENANT := "spinnaker"
endif

ROOT_DIR        := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
PKG             := github.com/oracle/mysql-operator
REGISTRY        := iad.ocir.io/$(TENANT)
SRC_DIRS        := cmd pkg test/examples

ARCH    := amd64
OS      := linux

.PHONY: all
all: build

.PHONY: test
test: build-dirs Makefile
	@echo "Testing: $(SRC_DIRS)"
	PKG=$(PKG) ./hack/test.sh $(SRC_DIRS)

.PHONY: build-dirs
build-dirs:
	@echo "Creating build directories"
	@mkdir -p bin/$(OS)_$(ARCH)
	@mkdir -p dist/

.PHONY: dist
dist: build-dirs
	@echo "Creating version file: $(VERSION)"
	@echo ${VERSION} > dist/version.txt

.PHONY: build
build: dist build-dirs Makefile
	@touch pkg/version/version.go # Important. Work around for https://github.com/golang/go/issues/18369
	@echo "Building mysql-operator"
	@GOOS=${OS} GOARCH=${ARCH} go build -i -v -o bin/mysql-operator -installsuffix "static" \
	    -ldflags "-X main.version=${VERSION} -X main.build=${BUILD}" \
	    ./cmd/mysql-operator/
	@echo "Building mysql-agent"
	@GOOS=${OS} GOARCH=${ARCH} go build -i -v -o bin/mysql-agent -installsuffix "static" \
	    -ldflags "-X main.version=${VERSION} -X main.build=${BUILD}" \
	    ./cmd/mysql-agent/

# Note: Only used for development, i.e. in CI the images are pushed using Wercker.
.PHONY: push
push:
	@docker build --build-arg=http_proxy --build-arg=https_proxy -t $(REGISTRY)/mysql-operator:$(VERSION) -f docker/mysql-operator/Dockerfile .
	@docker build --build-arg=http_proxy --build-arg=https_proxy -t $(REGISTRY)/mysql-agent:$(VERSION) -f docker/mysql-agent/Dockerfile .
	@docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY)
	@docker push $(REGISTRY)/mysql-operator:$(VERSION)
	@docker push $(REGISTRY)/mysql-agent:$(VERSION)

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: lint
lint:
	@find pkg cmd -name '*.go' | grep -v 'generated' | xargs -L 1 golint

.PHONY: clean
clean: rm -rf bin

.PHONY: run-dev
run-dev:
	@go run \
	    -ldflags "-X ${PKG}/pkg/version.buildVersion=${MYSQL_AGENT_VERSION}" \
	    cmd/mysql-operator/main.go \
	    --mysql-agent-image=iad.ocir.io/spinnaker/mysql-agent \
	    --kubeconfig=${KUBECONFIG} \
	    --v=4 \
	    --namespace=${USER}

.PHONY: generate
generate:
	./hack/update-generated.sh
