USE_GLOBAL_NAMESPACE ?= false

ifdef WERCKER
    # Insert swear words about mysql group replication and hostname length. Arghh..
    NEW_NAMESPACE ?= e2e-$(shell echo ${WERCKER_GIT_COMMIT} | fold -w 8 | head -n1)
    VERSION := ${WERCKER_GIT_COMMIT}
    E2E_FUNC := e2efunc-wercker
else
    NEW_NAMESPACE ?= e2e-${USER}
    VERSION := ${USER}-$(shell  date +%Y%m%d%H%M%S)
    E2E_FUNC := e2efunc-docker
endif

ROOT_DIR        := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
PKG             := github.com/oracle/mysql-operator
REGISTRY        := wcr.io/oracle
SRC_DIRS        := cmd pkg
TEST_E2E_IMAGE  := wcr.io/oracle/mysql-operator-gitlab-ci-e2e:1.0.0
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
	$(error "KUBECONFIG or KUBCONFIG_VAR must be defined")
else
	$(eval KUBECONFIG:=/tmp/kubeconf-$(shell date +'%d%m%y%H%M%S%N').conf)
	$(eval export KUBECONFIG)
	$(shell echo "$${KUBECONFIG_VAR}" | openssl enc -base64 -d -A > $(KUBECONFIG))
endif
endif

ifndef S3_UPLOAD_CREDS
ifndef S3_UPLOAD_CREDS_VAR
	$(error "S3_UPLOAD_CREDS or S3_UPLOAD_CREDS_VAR must be defined")
else
	$(eval S3_UPLOAD_CREDS:=/tmp/s3_upload_creds-$(shell date +'%d%m%y%H%M%S%N').yaml)
	$(eval export S3_UPLOAD_CREDS)
	$(shell echo "$${S3_UPLOAD_CREDS_VAR}" | openssl enc -base64 -d -A > $(S3_UPLOAD_CREDS))
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
	   -e USE_GLOBAL_NAMESPACE=$(USE_GLOBAL_NAMESPACE)                               \
	   -e NEW_NAMESPACE=$(NEW_NAMESPACE)                                             \
	   -e HOME=/tmp                                                                  \
	   $(TEST_E2E_IMAGE)                                                             \
	   /bin/sh -c "./test/e2e/scripts/e2e-mysql-operator-cluster.sh $(1)"
endef

e2e-setup-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), setup)

e2e-run-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), run)

e2e-teardown-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), teardown)

e2e-%: build-dirs e2econfig
	export E2E_TEST_RUN=$* && $(call $(E2E_FUNC), teardown setup run teardown)

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
	    --namespace=default

.PHONY: generate
generate:
	./hack/update-generated.sh

print-var-%: e2econfig
	@echo $* = $($*)

.PHONY: precommit-install
precommit-install:
	ln -s ${ROOT_DIR}/hack/pre-commit.sh ${ROOT_DIR}/.git/hooks/pre-commit
