package collector

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mrlhansen/idrac_exporter/internal/config"
)

func TestRedfishCertificateVerification(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	config.Config = &config.RootConfig{Timeout: 5, Concurrency: 1}
	host := strings.TrimPrefix(server.URL, "https://")

	signer := x509.NewCertPool()
	signer.AddCert(server.Certificate())

	// Every httptest server shares one certificate, so the unrelated CA is made here.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "unrelated CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	unrelatedCert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	unrelated := x509.NewCertPool()
	unrelated.AddCert(unrelatedCert)

	cases := []struct {
		name       string
		skipVerify bool
		rootCAs    *x509.CertPool
		wantErr    bool
	}{
		{"skipped verification accepts any certificate", true, nil, false},
		{"system roots refuse the test certificate", false, nil, true},
		{"ca_file that signed the target", false, signer, false},
		{"ca_file that did not sign the target", false, unrelated, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auth := &config.AuthConfig{Scheme: "https", HTTP: config.HttpConfig{SkipVerify: tc.skipVerify, RootCAs: tc.rootCAs}}
			r := NewRedfish(host, auth)

			resp, err := r.http.Get(r.baseurl + redfishRootPath)
			if resp != nil {
				resp.Body.Close()
			}
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v, want error %v", err, tc.wantErr)
			}
		})
	}
}
