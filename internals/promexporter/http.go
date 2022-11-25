package promexporter

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	contentTypeHeader     = "Content-Type"
	contentEncodingHeader = "Content-Encoding"
	acceptEncodingHeader  = "Accept-Encoding"
)

var gzipPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

func MetricsHandler(metricsPrefix string) func(http.ResponseWriter, *http.Request) {
	return func(rsp http.ResponseWriter, req *http.Request) {
		target := req.URL.Query().Get("target")
		if target == "" {
			http.Error(rsp, "Query parameter 'target' is mandatory", http.StatusBadRequest)
			return
		}

		c, err := getCollector(target, metricsPrefix)
		if err != nil {
			errorMsg := fmt.Sprintf("Error instantiating metrics collector for host %s: %v\n", target, err)
			log.Println(errorMsg)
			http.Error(rsp, errorMsg, http.StatusInternalServerError)
			return
		}

		metrics, err := c.CollectMetrics()
		if err != nil {
			errorMsg := fmt.Sprintf("Error collecting metrics for host %s: %v\n", target, err)
			log.Println(errorMsg)
			http.Error(rsp, errorMsg, http.StatusInternalServerError)
			return
		}

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
