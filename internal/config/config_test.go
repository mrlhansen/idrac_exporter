package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeTestCA writes a self-signed CA certificate as PEM and returns its path.
func writeTestCA(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
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
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateCertificateVerification(t *testing.T) {
	caFile := writeTestCA(t)
	skip, verify := true, false

	cases := []struct {
		name           string
		insecure       *bool
		caFile         string
		wantSkip       bool
		wantErr        string
		wantRootCAsSet bool
	}{
		{"unset skips verification", nil, "", true, "", false},
		{"false verifies against the system roots", &verify, "", false, "", false},
		{"false with ca_file verifies against it", &verify, caFile, false, "", true},
		{"unset with ca_file is refused", nil, caFile, true, "insecure_skip_verify: false", false},
		{"true with ca_file is refused", &skip, caFile, true, "insecure_skip_verify: false", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &AuthConfig{
				Username: "user",
				Password: "pass",
				HTTP:     HttpConfig{InsecureSkipVerify: tc.insecure, CAFile: tc.caFile},
			}
			err := c.validate(&RootConfig{})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got error %v, want one containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.HTTP.SkipVerify != tc.wantSkip {
				t.Errorf("SkipVerify = %v, want %v", c.HTTP.SkipVerify, tc.wantSkip)
			}
			if (c.HTTP.RootCAs != nil) != tc.wantRootCAsSet {
				t.Errorf("RootCAs set = %v, want %v", c.HTTP.RootCAs != nil, tc.wantRootCAsSet)
			}
		})
	}
}
