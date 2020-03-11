# Build cbif for entrypoint.
FROM golang:1.13 AS cbif-go-builder
ADD . /go/src/github.com/m-lab/gcp-config
#WORKDIR /go/src/github.com/m-lab/gcp-config/
RUN go get -v github.com/m-lab/gcp-config/cmd/cbif

# Build Go version of jsonnet.
FROM golang:1.13 AS jsonnet-go-builder
RUN apt-get install -y git
RUN go get -v github.com/google/go-jsonnet/cmd/jsonnet

# Build CPP version of jsonnet.
# NOTE: Use the same base image as the final image so that shared libraries
# match the jsonnet binary.
FROM gcr.io/cloud-builders/gcloud AS jsonnet-cpp-builder
RUN apt-get update
RUN apt-get install -y build-essential git wget
WORKDIR /opt
RUN wget https://github.com/google/jsonnet/archive/v0.15.0.tar.gz
RUN tar -C /opt/ -xf v0.15.0.tar.gz
RUN mv jsonnet-0.15.0 jsonnet
RUN cd jsonnet && make

############################################################################
# FINAL IMAGE: based on upstream gcloud builder.
FROM gcr.io/cloud-builders/gcloud

# Install binaries from builds above.
COPY --from=cbif-go-builder  /go/bin/cbif /usr/bin/cbif
COPY --from=jsonnet-go-builder  /go/bin/jsonnet /usr/bin/jsonnet-go
COPY --from=jsonnet-cpp-builder /opt/jsonnet/jsonnet /usr/bin
COPY --from=jsonnet-cpp-builder /opt/jsonnet/jsonnetfmt /usr/bin
RUN curl -o /usr/bin/sjsonnet.jar https://github.com/lihaoyi/sjsonnet/releases/download/0.2.3/sjsonnet.jar
RUN chmod 755 /usr/bin/sjsonnet.jar

# Install additional dependencies.
RUN apt-get update
RUN apt-get install -y dnsutils ca-certificates default-jre-headless make
RUN update-ca-certificates

WORKDIR /
ENTRYPOINT ["/usr/bin/cbif"]
