package promexporter

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/mrlhansen/idrac_exporter/internals/logging"
)

const (
	contentTypeHeader     = "Content-Type"
	contentEncodingHeader = "Content-Encoding"
	acceptEncodingHeader  = "Accept-Encoding"
)

var logger = logging.NewLogger().Sugar()

var gzipPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

func MetricsHandler(metricsPrefix string) func(http.ResponseWriter, *http.Request) {
	return func(rsp http.ResponseWriter, req *http.Request) {
		target := req.URL.Query().Get("target")
		if target == "" {
			logger.Errorf("Received request from %s without 'target' parameter", req.Host)
			http.Error(rsp, "Query parameter 'target' is mandatory", http.StatusBadRequest)
			return
		}

		logger.Debugf("Handling request from %s for host %s", req.Host, target)

		c, err := getCollector(target, metricsPrefix)
		if err != nil {
			errorMsg := fmt.Sprintf("Error instantiating metrics collector for host %s: %v\n", target, err)
			logger.Error(errorMsg)
			http.Error(rsp, errorMsg, http.StatusInternalServerError)
			return
		}

		logger.Debugf("Collecting metrics for host %s", target)

		metrics, err := c.CollectMetrics()
		if err != nil {
			errorMsg := fmt.Sprintf("Error collecting metrics for host %s: %v\n", target, err)
			logger.Error(errorMsg)
			http.Error(rsp, errorMsg, http.StatusInternalServerError)
			return
		}

		logger.Debugf("Metrics for host %s collected", target)

		header := rsp.Header()
		header.Set(contentTypeHeader, "text/plain")

		// Code inspired by the official Prometheus metrics http handler
		w := io.Writer(rsp)
		if gzipAccepted(req.Header) {
			header.Set(contentEncodingHeader, "gzip")
			gz := gzipPool.Get().(*gzip.Writer)
			defer gzipPool.Put(gz)

			gz.Reset(w)
			defer gz.Close()

			w = gz
		}

		_, _ = fmt.Fprint(w, metrics)
	}
}

// gzipAccepted returns whether the client will accept gzip-encoded content.
func gzipAccepted(header http.Header) bool {
	a := header.Get(acceptEncodingHeader)
	parts := strings.Split(a, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			return true
		}
	}
	return false
}
