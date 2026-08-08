package http

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestTrustedHTTPSAttestorPlaintextTrustedPeerRequiresOneExactHeader(t *testing.T) {
	attestor, err := NewTrustedHTTPSAttestor([]string{"10.0.0.0/8", "2001:db8::/32"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, peer string
		headers    []string
		want       bool
	}{
		{"trusted exact", "10.1.2.3:1234", []string{"https"}, true},
		{"trusted missing", "10.1.2.3:1234", nil, false},
		{"untrusted spoof", "198.51.100.9:1234", []string{"https"}, false},
		{"http", "10.1.2.3:1234", []string{"http"}, false},
		{"uppercase", "10.1.2.3:1234", []string{"HTTPS"}, false},
		{"whitespace", "10.1.2.3:1234", []string{" https"}, false},
		{"comma", "10.1.2.3:1234", []string{"https, https"}, false},
		{"duplicate", "10.1.2.3:1234", []string{"https", "https"}, false},
		{"mixed duplicate", "10.1.2.3:1234", []string{"https", "http"}, false},
		{"uri", "10.1.2.3:1234", []string{"https://example.com"}, false},
		{"ipv6", "[2001:db8::9]:1234", []string{"https"}, true},
		{"mapped ipv4", "[::ffff:10.1.2.3]:1234", []string{"https"}, true},
		{"malformed peer", "not-a-socket", []string{"https"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(ctx *fiber.Ctx) error {
				ctx.Context().SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(hostPart(test.peer)), Port: 1234})
				if test.name == "malformed peer" {
					ctx.Context().SetRemoteAddr(stringAddr(test.peer))
				}
				for _, header := range test.headers {
					ctx.Context().Request.Header.Add(forwardedProtoHeader, header)
				}
				return ctx.SendString(strconv.FormatBool(attestor.OriginalRequestWasHTTPS(ctx)))
			})
			request := httptest.NewRequest(nethttp.MethodGet, "http://example.test/", nil)
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 200 {
				t.Fatalf("status = %d", response.StatusCode)
			}
			if response.Header.Get("Content-Length") == "0" {
				t.Fatal("attestation response was empty")
			}
			if responseBody(t, response) != strconv.FormatBool(test.want) {
				t.Fatalf("attestation did not equal %v", test.want)
			}
		})
	}
}

func responseBody(t *testing.T, response *nethttp.Response) string {
	t.Helper()
	defer response.Body.Close()
	buffer, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(buffer)
}

func TestTrustedHTTPSAttestorAcceptsActualDirectTLSRegardlessOfForwardedHeader(t *testing.T) {
	attestor, err := NewTrustedHTTPSAttestor(nil)
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		if !attestor.OriginalRequestWasHTTPS(ctx) {
			t.Fatal("actual TLS was not attested")
		}
		return ctx.SendStatus(204)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	certificate := testCertificate(t)
	tlsListener := tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13})
	done := make(chan error, 1)
	go func() { done <- app.Listener(tlsListener) }()
	t.Cleanup(func() {
		_ = app.Shutdown()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("TLS Fiber server did not stop")
		}
	})
	client := &nethttp.Client{Transport: &nethttp.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS13}}}
	request, err := nethttp.NewRequest(nethttp.MethodGet, "https://"+listener.Addr().String()+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add(forwardedProtoHeader, "https")
	request.Header.Add(forwardedProtoHeader, "http")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func hostPart(value string) string {
	host, _, err := net.SplitHostPort(value)
	if err != nil {
		return "127.0.0.1"
	}
	return host
}

type stringAddr string

func (address stringAddr) Network() string { return "tcp" }
func (address stringAddr) String() string  { return string(address) }

func testCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"localhost"}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, public, private)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	privateDER, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
	certificate, err := tls.X509KeyPair(certificatePEM, privatePEM)
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}
