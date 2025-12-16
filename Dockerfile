FROM golang:1.24-alpine3.20 AS builder

WORKDIR /app/src
RUN apk add -U make git grep
COPY . .
RUN make build

FROM alpine:3.20 AS container

WORKDIR /app
COPY --from=builder /app/src/idrac_exporter /app/bin/
RUN apk --no-cache upgrade
COPY default-config.yml /etc/prometheus/idrac.yml
COPY entrypoint.sh /app
ENTRYPOINT ["/app/entrypoint.sh"]
