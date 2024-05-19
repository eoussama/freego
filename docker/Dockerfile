FROM golang:1.22-alpine

RUN apk add --no-cache bash

ENV GOPATH=/go
ENV GO111MODULE=on
ENV PRJECT_PATH=$GOPATH/src/github.com/eoussama/freego
WORKDIR $PRJECT_PATH

COPY .env .
COPY ./src .
COPY ./scripts/entrypoint.sh .

RUN go mod tidy
RUN source .env

CMD ["bash"]
VOLUME ["/$PRJECT_PATH"]
ENTRYPOINT ["./entrypoint.sh"]