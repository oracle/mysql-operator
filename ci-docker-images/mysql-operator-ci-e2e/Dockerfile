FROM oraclelinux:7.3

RUN yum install -y ca-certificates make openssl git && yum clean all

# Install golang environment
RUN curl https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz -O && \
mkdir /tools && \
tar xzf go1.8.3.linux-amd64.tar.gz -C /tools && \
rm go1.8.3.linux-amd64.tar.gz && \
mkdir /go

RUN mkdir /go/bin
ENV PATH=/tools/go/bin:$PATH
ENV GOPATH=/go
ENV GOROOT=/tools/go

RUN curl -L -O https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-linux-amd64.tar.gz && \
tar xf glide-v0.12.3-linux-amd64.tar.gz -C /tools

ENV PATH=/tools/linux-amd64:$PATH

# Install the kubectl client
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.8.4/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin/kubectl

# Installs Helm
RUN curl -LO https://kubernetes-helm.storage.googleapis.com/helm-v2.6.1-linux-amd64.tar.gz
RUN tar -xzvf helm-v2.6.1-linux-amd64.tar.gz
RUN mv linux-amd64/helm /usr/local/bin/helm
