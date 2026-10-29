FROM golang:1.27-alpine3.24 AS builder

WORKDIR /app/src
RUN apk add --no-cache make git grep
COPY . .
RUN make build

FROM alpine:3.24 AS container

WORKDIR /app
COPY --from=builder /app/src/idrac_exporter /app/bin/
RUN apk upgrade --no-cache && \
    apk add --no-cache bash
COPY default-config.yml /etc/prometheus/idrac.yml
COPY entrypoint.sh /app
ENTRYPOINT ["/app/entrypoint.sh"]
