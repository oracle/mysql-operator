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
    VERSION ?= ${USER}-$(shell git describe --always --dirty)
    TENANT ?= "spinnaker"
endif

PKG             := github.com/oracle/mysql-operator
REGISTRY        := iad.ocir.io
SRC_DIRS        := cmd pkg test/examples
CMD_DIRECTORIES := $(sort $(dir $(wildcard ./cmd/*/)))
COMMANDS        := $(CMD_DIRECTORIES:./cmd/%/=%)

ARCH    ?= amd64
OS      ?= linux
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

.PHONY: build-docker
build-docker:
	@docker build \
    	--build-arg=http_proxy \
    	--build-arg=https_proxy \
    	-t $(REGISTRY)/$(TENANT)/mysql-operator:$(VERSION) \
    	-f docker/mysql-operator/Dockerfile .

	@docker build \
	--build-arg=http_proxy \
	--build-arg=https_proxy \
	-t $(REGISTRY)/$(TENANT)/mysql-agent:$(VERSION) \
	-f docker/mysql-agent/Dockerfile .

# Note: Only used for development, i.e. in CI the images are pushed using Wercker.
.PHONY: push
push: build build-docker
	@docker login iad.ocir.io -u $(DOCKER_REGISTRY_USERNAME) -p '$(DOCKER_REGISTRY_PASSWORD)'
	@docker push $(REGISTRY)/$(TENANT)/mysql-operator:$(VERSION)
	@docker push $(REGISTRY)/$(TENANT)/mysql-agent:$(VERSION)

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: lint
lint:
	@find pkg cmd -name '*.go' | grep -v 'generated' | xargs -L 1 golint

.PHONY: clean
clean:
	rm -rf .go bin

.PHONY: run-dev
run-dev:
	@go run \
	    -ldflags "-X ${PKG}/pkg/version.buildVersion=${MYSQL_AGENT_VERSION}" \
	    cmd/mysql-operator/main.go \
	    --mysql-agent-image=iad.ocir.io/$(TENANT)/mysql-agent \
	    --kubeconfig=${KUBECONFIG} \
	    --v=4 \
	    --namespace=${USER}

.PHONY: generate
generate:
	./hack/update-generated.sh
