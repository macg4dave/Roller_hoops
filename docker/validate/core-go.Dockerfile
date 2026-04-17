FROM golang:1.24-alpine AS base
WORKDIR /workspace/core-go

COPY core-go/go.mod core-go/go.sum ./
RUN go mod download

COPY api /workspace/api
COPY core-go/ ./

FROM base AS fmtcheck
RUN test -z "$(gofmt -l .)"

FROM base AS vet
RUN go vet ./...

FROM base AS test
RUN go test ./...
