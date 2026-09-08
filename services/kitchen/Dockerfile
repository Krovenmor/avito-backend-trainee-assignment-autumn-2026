FROM golang:1.26.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/kitchen ./services/kitchen/cmd/app

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 kitchen

COPY --from=builder /out/kitchen /usr/local/bin/kitchen

USER kitchen

EXPOSE 8080

ENTRYPOINT ["kitchen"]
