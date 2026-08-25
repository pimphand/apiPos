FROM golang:1.24-alpine AS build

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/server

FROM alpine:3.20.1 AS prod
WORKDIR /app
COPY --from=build /app/main /app/main
COPY --from=build /app/db /app/db
COPY --from=build /app/docs /app/docs
COPY --from=build /app/gen /app/gen
EXPOSE 50051 8080
CMD ["./main"]
