# Development image with Go and smee-client, for receiving webhooks locally.
# Build and run it with scripts/start.sh.
FROM golang:1.22-alpine

RUN apk add --no-cache bash nodejs npm \
  && npm install --global smee-client

ENV PROJECT_PATH=/go/src/github.com/eoussama/freego
WORKDIR $PROJECT_PATH

# .env is excluded by .dockerignore; it is read at runtime from the mounted
# project directory instead of being baked into the image.
COPY . .
RUN go mod download

EXPOSE 8080

VOLUME ["$PROJECT_PATH"]
ENTRYPOINT ["./scripts/entrypoint.sh"]
CMD ["bash"]
