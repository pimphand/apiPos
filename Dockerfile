FROM golang:1.24-alpine AS build

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Cache-busting: force rebuild when source changes
ARG CACHEBUST=1
RUN echo "Cache bust: ${CACHEBUST}"

COPY . .

# Bust Go build cache to ensure fresh compilation
ENV GOCACHE=/tmp/go-cache
RUN go clean -cache && go build -o main ./cmd/server

# Install the golang-migrate CLI so migrations can be run inside the container
# (e.g. `make migrate-up` after `docker compose exec pos-app sh`)
# NOTE: the postgres build tag is required, otherwise the CLI reports
# "database driver: unknown driver postgres (forgotten import?)"
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

FROM alpine:3.20.1 AS prod
WORKDIR /app
COPY --from=build /app/main /app/main
COPY --from=build /app/db /app/db
COPY --from=build /app/docs /app/docs
COPY --from=build /app/gen /app/gen
COPY --from=build /go/bin/migrate /usr/local/bin/migrate
# make + Makefile so database commands work inside the container
COPY Makefile /app/Makefile
RUN apk add --no-cache make
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
EXPOSE 50051 8080
ENTRYPOINT ["/app/entrypoint.sh"]
