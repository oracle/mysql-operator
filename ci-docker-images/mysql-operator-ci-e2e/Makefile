IMAGE ?= iad.ocir.io/oracle/mysql-operator-ci-e2e
VERSION ?= 1.1.1

.PHONY: build
build:
	docker build \
	  --build-arg=http_proxy \
	  --build-arg=https_proxy \
	  -f Dockerfile \
	  -t ${IMAGE}:${VERSION} \
	  .

.PHONY: push
push: build
	docker push ${IMAGE}:${VERSION}
