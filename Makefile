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
    NEW_NAMESPACE ?= e2e-$(shell echo ${WERCKER_GIT_COMMIT} | fold -w 8 | head -n1)
    VERSION := ${WERCKER_GIT_COMMIT}
    E2E_FUNC := e2efunc-wercker
    E2E_NON_BUFFERED_LOGS ?= false
else
    NEW_NAMESPACE ?= e2e-${USER}
    VERSION := ${USER}-$(shell date +%Y%m%d%H%M%S)
    E2E_FUNC := e2efunc-docker
    E2E_NON_BUFFERED_LOGS ?= true
endif

E2E_PARALLEL    ?= 10
ROOT_DIR        := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
PKG             := github.com/oracle/mysql-operator
REGISTRY        := iad.ocir.io/spinnaker
SRC_DIRS        := cmd pkg test/examples
TEST_E2E_IMAGE  := iad.ocir.io/oracle/mysql-operator-ci-e2e:1.0.0
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

.PHONY:e2econfig
e2econfig:
ifndef KUBECONFIG
ifndef KUBECONFIG_VAR
	$(error "KUBECONFIG or KUBECONFIG_VAR must be defined")
else
	$(eval KUBECONFIG:=/tmp/kubeconf-$(shell date +'%d%m%y%H%M%S%N').conf)
	$(eval export KUBECONFIG)
	$(shell echo "$${KUBECONFIG_VAR}" | openssl enc -base64 -d -A > $(KUBECONFIG))
endif
endif

ifndef CLUSTER_INSTANCE_SSH_KEY
ifndef CLUSTER_INSTANCE_SSH_KEY_VAR
	$(error "CLUSTER_INSTANCE_SSH_KEY or CLUSTER_INSTANCE_SSH_KEY_VAR must be defined")
else
	$(eval CLUSTER_INSTANCE_SSH_KEY:=/tmp/cluster_instance_key)
	$(eval export CLUSTER_INSTANCE_SSH_KEY)
	$(shell echo "$${CLUSTER_INSTANCE_SSH_KEY_VAR}" | openssl enc -base64 -d -A > $(CLUSTER_INSTANCE_SSH_KEY))
	$(shell chmod 600 $(CLUSTER_INSTANCE_SSH_KEY))
endif
endif

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

define e2efunc-wercker
	if [ -z "$$MYSQL_OPERATOR_VERSION" ]; then export MYSQL_OPERATOR_VERSION=`cat dist/version.txt`; fi && \
	export NEW_NAMESPACE=$(NEW_NAMESPACE) && \
	export USE_GLOBAL_NAMESPACE=$(USE_GLOBAL_NAMESPACE) && \
	export E2E_NON_BUFFERED_LOGS=$(E2E_NON_BUFFERED_LOGS) && \
	export E2E_PARALLEL=$(E2E_PARALLEL) && \
	./test/e2e/scripts/e2e-mysql-operator-cluster.sh $(1)
endef

define e2efunc-docker
	   if [ -z "$$MYSQL_OPERATOR_VERSION" ]; then export MYSQL_OPERATOR_VERSION=`cat dist/version.txt`; fi && \
	   docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY) && \
	   docker run                                                                    \
	   ${DOCKER_OPS_INTERACTIVE}                                                     \
	   --rm                                                                          \
	   -v "$$(pwd)/.go:/go:delegated"                                                \
	   -v "$$(pwd):/go/src/$(PKG):delegated"                                         \
	   -v "$$(pwd)/bin/$(ARCH):/go/bin"                                              \
	   -v "$$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static:delegated" \
	   -w /go/src/$(PKG)                                                             \
	   -e CLUSTER_INSTANCE_SSH_KEY=$(CLUSTER_INSTANCE_SSH_KEY)                       \
	   -v $(CLUSTER_INSTANCE_SSH_KEY):$(CLUSTER_INSTANCE_SSH_KEY)                    \
	   -e KUBECONFIG=/kubeconfig.conf                                                \
	   -v $(KUBECONFIG):/kubeconfig.conf                                             \
	   -e S3_ACCESS_KEY=$(S3_ACCESS_KEY)                                             \
	   -e S3_SECRET_KEY=$(S3_SECRET_KEY)                                             \
	   -e E2E_DEBUG="$(E2E_DEBUG)"                                                   \
	   -e USE_RBAC="$(USE_RBAC)"                                                     \
	   -e NODE_IPS="$(NODE_IPS)"                                                     \
	   -e DOCKER_REGISTRY_USERNAME="$$DOCKER_REGISTRY_USERNAME"                      \
	   -e DOCKER_REGISTRY_PASSWORD="$$DOCKER_REGISTRY_PASSWORD"                      \
	   -e MYSQL_OPERATOR_VERSION="$$MYSQL_OPERATOR_VERSION"                          \
	   -e HTTP_PROXY="$$HTTP_PROXY"                                                  \
	   -e HTTPS_PROXY="$$HTTPS_PROXY"                                                \
	   -e NO_PROXY="$$NO_PROXY"                                                      \
	   -e E2E_TEST_RUN="$$E2E_TEST_RUN"                                              \
	   -e E2E_TEST_TAG="$$E2E_TEST_TAG"                                              \
	   -e USE_GLOBAL_NAMESPACE=$(USE_GLOBAL_NAMESPACE)                               \
	   -e NEW_NAMESPACE=$(NEW_NAMESPACE)                                             \
	   -e E2E_NON_BUFFERED_LOGS=$(E2E_NON_BUFFERED_LOGS)                             \
	   -e E2E_PARALLEL=$(E2E_PARALLEL)                                               \
	   -e HOME=/tmp                                                                  \
	   $(TEST_E2E_IMAGE)                                                             \
	   /bin/sh -c "./test/e2e/scripts/e2e-mysql-operator-cluster.sh $(1)"
endef

# Runs test set specified by regex (i.e. go test -run <regex>)

e2e-test-setup-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), setup)

e2e-test-run-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), run)

e2e-test-teardown-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), teardown)

e2e-test-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), teardown setup run teardown)

# Runs test set specified by tags (i.e. go test -tags <tag>)

e2e-suite-setup-%: build-dirs e2econfig
	export E2E_TEST_TAG=$* && $(call $(E2E_FUNC), setup)

e2e-suite-run-%: build-dirs e2econfig
	export E2E_TEST_TAG=$* && $(call $(E2E_FUNC), run)

e2e-suite-teardown-%: build-dirs e2econfig
	export E2E_TEST_TAG=$* && $(call $(E2E_FUNC), teardown)

e2e-suite-%: build-dirs e2econfig
	export E2E_TEST_TAG=$* && $(call $(E2E_FUNC), teardown setup run teardown)

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

print-var-%: e2econfig
	@echo $* = $($*)

.PHONY: precommit-install
precommit-install:
	ln -s ${ROOT_DIR}/hack/pre-commit.sh ${ROOT_DIR}/.git/hooks/pre-commit
