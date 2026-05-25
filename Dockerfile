# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
ARG SERVICE=global
RUN CGO_ENABLED=0 go build -o /out/service ./cmd/${SERVICE}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=build /out/service /app/service
EXPOSE 8080
ENTRYPOINT ["/app/service"]
