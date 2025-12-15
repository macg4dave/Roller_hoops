FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -o /out/core-go ./cmd/core-go

FROM alpine:3.20
RUN apk add --no-cache iputils libcap nmap && \
    (setcap cap_net_raw+ep /bin/ping || true)
RUN adduser -D -H -s /sbin/nologin app
USER app
WORKDIR /app

COPY --from=build /out/core-go /app/core-go
EXPOSE 8081

ENV HTTP_ADDR=:8081
CMD ["/app/core-go"]
