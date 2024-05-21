FROM golang:1.22-alpine

RUN apk add --no-cache bash nodejs npm
RUN npm install --global smee-client

ENV GOPATH=/go
ENV GO111MODULE=on
ENV PRJECT_PATH=$GOPATH/src/github.com/eoussama/freego
WORKDIR $PRJECT_PATH

COPY .env .
COPY ./src .
COPY ./go.* .
COPY ./main.go .
COPY ./scripts/smee.sh .
COPY ./scripts/entrypoint.sh .

RUN go mod tidy
RUN source .env

EXPOSE 8080

CMD ["bash"]
VOLUME ["/$PRJECT_PATH"]
ENTRYPOINT ["./scripts/entrypoint.sh"]