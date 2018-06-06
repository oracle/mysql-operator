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

USE_GLOBAL_NAMESPACE ?= false

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
REGISTRY_STRING := $(subst /,_,$(REGISTRY))
CMD_DIRECTORIES := $(sort $(dir $(wildcard ./cmd/*/)))
COMMANDS        := $(CMD_DIRECTORIES:./cmd/%/=%)
CONTAINER_FILES := $(addprefix .container-$(REGISTRY_STRING)-,$(addsuffix -$(VERSION),$(COMMANDS)))
PUSH_FILES      := $(addprefix .push-$(REGISTRY_STRING)-,$(addsuffix -$(VERSION),$(COMMANDS)))

ARCH    := amd64
OS      := linux
UNAME_S := $(shell uname -s)

ifeq ($(UNAME_S),Darwin)
	# Cross-compiling from OSX to linux, go install puts the binaries in $GOPATH/bin/$GOOS_$GOARCH
    BINARIES := $(addprefix $(GOPATH)/bin/$(OS)_$(ARCH)/,$(COMMANDS))
else
ifeq ($(UNAME_S),Linux)
	# Compiling on linux for linux, go install puts the binaries in $GOPATH/bin
    BINARIES := $(addprefix $(GOPATH)/bin/,$(COMMANDS))
else
	$(error "Unsupported OS: $(UNAME_S)")
endif
endif

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
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

.PHONY: dist
dist: build-dirs
	@echo "Creating version file: $(VERSION)"
	@echo ${VERSION} > dist/version.txt

.PHONY: build
build: dist build-dirs Makefile
	@echo "Building: $(BINARIES)"
	@touch pkg/version/version.go # Important. Work around for https://github.com/golang/go/issues/18369
	ARCH=$(ARCH) OS=$(OS) VERSION=$(VERSION) PKG=$(PKG) ./hack/build.sh
	cp $(BINARIES) ./bin/$(OS)_$(ARCH)/

# Note: Only used for development, i.e. in CI the images are built using Wercker.
.PHONY: containers
containers: $(CONTAINER_FILES)
.container-$(REGISTRY_STRING)-%-$(VERSION): build dist
	@echo Builing container: $*
	@docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY)
	@docker build --build-arg=http_proxy --build-arg=https_proxy -t $(REGISTRY)/$*:$(VERSION) -f docker/$*/Dockerfile .
	@docker images -q $(REGISTRY)/$*:$(VERSION) > $@

# Note: Only used for development, i.e. in CI the images are pushed using Wercker.
.PHONY: push
push: $(PUSH_FILES)
.push-$(REGISTRY_STRING)-%-$(VERSION): .container-$(REGISTRY_STRING)-%-$(VERSION)
	@echo Pushing container: $*
	@docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY)
	@docker push $(REGISTRY)/$*:$(VERSION)
	@docker images -q $(REGISTRY)/$*:$(VERSION) > $@

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: lint
lint:
	@find pkg cmd -name '*.go' | grep -v 'generated' | xargs -L 1 golint

.PHONY: clean
clean: container-clean bin-clean

.PHONY: container-clean
container-clean:
	rm -rf .container-* .push-* dist

.PHONY: bin-clean
bin-clean:
	rm -rf .go bin

.PHONY: run-dev
run-dev:
	@go run \
	    -ldflags "-X ${PKG}/pkg/version.buildVersion=${MYSQL_AGENT_VERSION}" \
	    cmd/mysql-operator/main.go \
	    --kubeconfig=${KUBECONFIG} \
	    --v=4 \
	    --namespace=${USER}

.PHONY: generate
generate:
	./hack/update-generated.sh
