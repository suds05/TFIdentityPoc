# syntax=docker/dockerfile:1
################################################################
# 
# Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
# Licensed under the Apache License, Version 2.0 (the "License")
# 
# Multi-stage Docker build for global or storage tier Go services.
#
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
