package util

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
)

// NewHTTPClient returns an http client with TLS support if needed.
func NewHTTPClient(useTLS bool) (*http.Client, error) {
	if useTLS {
		// Tries to find system's certificate pool
		caCertPool, _ := x509.SystemCertPool() // #nosec - SystemCertPool tries to get local cert pool, if it fails, a new cert pool is created
		if caCertPool == nil {
			caCertPool = x509.NewCertPool()
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:               tls.VersionTLS12,
					MaxVersion:               tls.VersionTLS13,
					PreferServerCipherSuites: true,
					InsecureSkipVerify:       false,
					RootCAs:                  caCertPool,
				},
			},
		}
		return client, nil
	}

	client := &http.Client{}
	return client, nil
}

// IsHTTPS checks if a URL uses HTTPS
func IsHTTPS(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "https://")
}
