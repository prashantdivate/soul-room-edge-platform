package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unified-fleet/edge-agent/internal/protocol"
)

func main() {
	var addr, state string
	flag.StringVar(&addr, "addr", "127.0.0.1:9443", "listen address")
	flag.StringVar(&state, "state", ".dev-control-plane", "state directory")
	flag.Parse()
	s, err := newServer(addr, state)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("device simulator listening on https://%s", addr)
	log.Fatal(s.ListenAndServeTLS("", ""))
}

type server struct {
	*http.Server
	state string
	mu    sync.Mutex
	used  map[string]bool
	caPEM []byte
	caKey ed25519.PrivateKey
}

func newServer(addr, state string) (*server, error) {
	if err := os.MkdirAll(state, 0700); err != nil {
		return nil, err
	}
	cert, caPEM, caKey, err := loadOrCreateCerts(addr, state)
	if err != nil {
		return nil, err
	}
	s := &server{state: state, used: map[string]bool{}, caPEM: caPEM, caKey: caKey}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/enroll", s.enroll)
	mux.HandleFunc("/v1/heartbeat", acceptEnvelope("heartbeat"))
	mux.HandleFunc("/v1/telemetry", acceptEnvelope("telemetry"))
	mux.HandleFunc("/v1/inventory", acceptEnvelope("inventory"))
	mux.HandleFunc("/v1/offline", acceptEnvelope("offline"))
	s.Server = &http.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		},
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s, nil
}

func (s *server) enroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var req protocol.EnrollmentRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.used[req.Token] {
		s.mu.Unlock()
		http.Error(w, "token used", http.StatusForbidden)
		return
	}
	s.used[req.Token] = true
	s.mu.Unlock()
	deviceID := "dev-" + shortHash(req.PublicKeyPEM)
	certPEM, err := s.signClient(req.PublicKeyPEM, deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(protocol.EnrollmentResponse{
		TenantID: "dev-tenant", DeviceID: deviceID, ClientCertificatePEM: string(certPEM),
		CACertificatePEM: string(s.caPEM), ServerEndpoint: "https://" + r.Host,
	})
}

func acceptEnvelope(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var env protocol.Envelope
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&env); err != nil {
			http.Error(w, "bad envelope", http.StatusBadRequest)
			return
		}
		if err := env.Validate(time.Now(), 10*time.Minute); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted", "kind": kind, "message_id": env.MessageID})
	}
}

func (s *server) signClient(pubPEM, cn string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errInvalidPEM
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(ed25519.PublicKey)
	if !ok {
		return nil, errInvalidPEM
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caBlock, _ := pem.Decode(s.caPEM)
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, ca, pub, s.caKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), nil
}

var errInvalidPEM = &parseErr{"invalid public key pem"}

type parseErr struct{ s string }

func (e *parseErr) Error() string { return e.s }

func loadOrCreateCerts(addr, state string) (tls.Certificate, []byte, ed25519.PrivateKey, error) {
	caPath := filepath.Join(state, "ca.pem")
	caKeyPath := filepath.Join(state, "ca_key.pem")
	certPath := filepath.Join(state, "server.pem")
	keyPath := filepath.Join(state, "server_key.pem")
	if _, err := os.Stat(certPath); err == nil {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return tls.Certificate{}, nil, nil, err
		}
		caPEM, err := os.ReadFile(caPath)
		if err != nil {
			return tls.Certificate{}, nil, nil, err
		}
		keyPEM, err := os.ReadFile(caKeyPath)
		if err != nil {
			return tls.Certificate{}, nil, nil, err
		}
		caKey, err := parseEdKey(keyPEM)
		return cert, caPEM, caKey, err
	}
	caPub, caKey, _ := ed25519.GenerateKey(rand.Reader)
	caTpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "edge-agent-dev-ca"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(3650 * 24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, caPub, caKey)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	if err := os.WriteFile(caPath, caPEM, 0644); err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	if err := writeKey(caKeyPath, caKey); err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	srvPub, srvKey, _ := ed25519.GenerateKey(rand.Reader)
	host, _, _ := net.SplitHostPort(addr)
	serverTpl := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: host}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(365 * 24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, DNSNames: []string{"localhost"}}
	if ip := net.ParseIP(host); ip != nil {
		serverTpl.IPAddresses = []net.IP{ip}
	}
	der, err := x509.CreateCertificate(rand.Reader, serverTpl, caTpl, srvPub, caKey)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	if err := writeKey(keyPath, srvKey); err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	return cert, caPEM, caKey, err
}

func writeKey(path string, key ed25519.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600)
}

func parseEdKey(pemBytes []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errInvalidPEM
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ed, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errInvalidPEM
	}
	return ed, nil
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}
