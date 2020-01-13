FROM golang:1.13

ADD . /go/src/github.com/m-lab/gcp-config
RUN go get -v github.com/m-lab/gcp-config/cmd/stctl
RUN go get -v github.com/m-lab/gcp-config/cmd/cbif
ENV SINGLE_COMMAND true
ENTRYPOINT ["/go/bin/cbif"]
