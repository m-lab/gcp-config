FROM golang:1.20

ADD . /go/src/github.com/m-lab/gcp-config
WORKDIR /go/src/github.com/m-lab/gcp-config/
RUN go install -v ./cmd/stctl
RUN go install -v ./cmd/cbif
ENV SINGLE_COMMAND true
ENTRYPOINT ["/go/bin/cbif"]
