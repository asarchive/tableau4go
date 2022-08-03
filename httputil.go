package tableau4go

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
)

var (
	connectTimeOut   = 10 * time.Second
	readWriteTimeout = 20 * time.Second
)

func timeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(network, address string) (net.Conn, error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}

		if rwTimeout > 0 {
			if err = conn.SetDeadline(time.Now().Add(rwTimeout)); err != nil {
				return nil, err
			}
		}
		return conn, nil
	}
}

// apps will set two OS variables:
// atscale_http_sslcert - location of the http ssl cert
// atscale_http_sslkey - location of the http ssl key
func NewTimeoutClient(cTimeout time.Duration, rwTimeout time.Duration, useClientCerts bool) *http.Client {
	certLocation := os.Getenv("atscale_http_sslcert")
	keyLocation := os.Getenv("atscale_http_sslkey")
	caFile := os.Getenv("atscale_ca_file")

	// default tlsConfig
	//nolint:gosec // skip verify is currently allowed
	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	//nolint:nestif // TODO: simplify nested if's
	if useClientCerts && len(certLocation) > 0 && len(keyLocation) > 0 {
		// Load client cert if available
		if cert, loadKeyPairErr := tls.LoadX509KeyPair(certLocation, keyLocation); loadKeyPairErr == nil {
			if len(caFile) > 0 {
				caCertPool := x509.NewCertPool()
				caCert, err := ioutil.ReadFile(caFile)
				if err != nil {
					fmt.Printf("Error setting up caFile [%s]:%v\n", caFile, err)
				}
				caCertPool.AppendCertsFromPEM(caCert)

				//nolint:gosec // skip verify is currently allowed
				tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true, RootCAs: caCertPool}

				//nolint:staticcheck // SA1019 TODO: remove this line and let go negotiate the first matching cert
				tlsConfig.BuildNameToCertificate()
			} else {
				//nolint:gosec // skip verify is currently allowed
				tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
			}
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			Dial:            timeoutDialer(cTimeout, rwTimeout),
		},
	}
}

func DefaultTimeoutClient() *http.Client {
	return NewTimeoutClient(connectTimeOut, readWriteTimeout, false)
}
