ARG ARCH=
FROM ${ARCH}golang:1.19-alpine3.16 as builder

WORKDIR /app/src

COPY go.* ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN go build -o /app/bin/idrac_exporter ./cmd/idrac_exporter

FROM ${ARCH}alpine:3.16 as container

WORKDIR /app
COPY --from=builder /app/bin /app/bin
RUN apk add -U bash gettext
COPY idrac.yml.template /etc/prometheus/
COPY entrypoint.sh /app
ENTRYPOINT /app/entrypoint.sh
