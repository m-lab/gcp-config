FROM golang:1.20

ADD . /go/src/github.com/m-lab/gcp-config
WORKDIR /go/src/github.com/m-lab/gcp-config/
RUN go install -v github.com/m-lab/gcp-config/cmd/stctl@latest
RUN go install -v github.com/m-lab/gcp-config/cmd/cbif@latest
ENV SINGLE_COMMAND true
ENTRYPOINT ["/go/bin/cbif"]
