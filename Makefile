VERSION  = $(or $(shell git tag --points-at HEAD | grep -oP 'v\K[0-9.]+'), unknown)
REVISION = $(shell git rev-parse HEAD)

REPOSITORY := github.com/mrlhansen/idrac_exporter
LDFLAGS    := -w -s
LDFLAGS    += -X $(REPOSITORY)/internal/version.Version=$(VERSION)
LDFLAGS    += -X $(REPOSITORY)/internal/version.Revision=$(REVISION)
GOFLAGS    := -ldflags "$(LDFLAGS)"
RUNFLAGS   ?= -config config.yml -verbose

build:
	go build $(GOFLAGS) -o idrac_exporter ./cmd/idrac_exporter

run:
	go run ./cmd/idrac_exporter $(RUNFLAGS)
