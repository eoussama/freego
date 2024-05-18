FROM golang:1.22-alpine

RUN apk add --no-cache bash

ENV GOPATH=/go
ENV GO111MODULE=on
ENV PRJECT_PATH=$GOPATH/src/github.com/eoussama/freego
WORKDIR $PRJECT_PATH

COPY ./src .

RUN go mod tidy

CMD ["bash"]
VOLUME ["/$PRJECT_PATH"]