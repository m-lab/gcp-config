FROM golang:1.21

ADD . /go/src/github.com/m-lab/gcp-config
WORKDIR /go/src/github.com/m-lab/gcp-config/
RUN go install -v github.com/m-lab/gcp-config/cmd/stctl@v1.3.12
RUN go install -v github.com/m-lab/gcp-config/cmd/cbif@v1.3.12
ENV SINGLE_COMMAND true
ENTRYPOINT ["/go/bin/cbif"]
