FROM golang:1.26.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/testapi ./services/testapi/cmd/app

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 testapi

COPY --from=builder /out/testapi /usr/local/bin/testapi

USER testapi

EXPOSE 8081

ENTRYPOINT ["testapi"]
