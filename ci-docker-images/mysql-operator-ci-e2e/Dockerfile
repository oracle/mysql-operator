FROM oraclelinux:7.4

RUN yum install -y ca-certificates make openssl git && yum clean all

# Install golang environment
RUN curl https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz -O && \
    mkdir /tools && \
    tar xzf go1.8.3.linux-amd64.tar.gz -C /tools && \
    rm go1.8.3.linux-amd64.tar.gz && \
    mkdir -p /go/bin

ENV PATH=/tools/go/bin:/go/bin:/tools/linux-amd64:$PATH \
    GOPATH=/go \
    GOROOT=/tools/go

# Install the kubectl client
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.8.4/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

# Installs Helm
RUN curl -LO https://kubernetes-helm.storage.googleapis.com/helm-v2.6.1-linux-amd64.tar.gz && \
    tar -xzvf helm-v2.6.1-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm

# Install Ginkgo
RUN go get -u github.com/onsi/ginkgo/ginkgo
