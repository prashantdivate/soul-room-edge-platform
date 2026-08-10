package protocol

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const Version = "edge.v1"

type Envelope struct {
	ProtocolVersion      string          `json:"protocol_version"`
	MessageID            string          `json:"message_id"`
	TenantID             string          `json:"tenant_id,omitempty"`
	DeviceID             string          `json:"device_id,omitempty"`
	ParentGatewayID      string          `json:"parent_gateway_id,omitempty"`
	Sequence             uint64          `json:"sequence"`
	CreatedUnixNano      int64           `json:"created_unix_nano"`
	PayloadSchemaVersion string          `json:"payload_schema_version"`
	PayloadDigest        string          `json:"payload_digest"`
	Payload              json.RawMessage `json:"payload"`
}

func NewEnvelope(tenantID, deviceID, schema string, sequence uint64, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	sum := sha256.Sum256(raw)
	return Envelope{
		ProtocolVersion:      Version,
		MessageID:            newID(),
		TenantID:             tenantID,
		DeviceID:             deviceID,
		Sequence:             sequence,
		CreatedUnixNano:      time.Now().UnixNano(),
		PayloadSchemaVersion: schema,
		PayloadDigest:        hex.EncodeToString(sum[:]),
		Payload:              raw,
	}, nil
}

func (e Envelope) Validate(now time.Time, maxSkew time.Duration) error {
	if e.ProtocolVersion != Version {
		return fmt.Errorf("unsupported protocol version %q", e.ProtocolVersion)
	}
	if e.MessageID == "" {
		return fmt.Errorf("missing message_id")
	}
	created := time.Unix(0, e.CreatedUnixNano)
	if created.After(now.Add(maxSkew)) || created.Before(now.Add(-maxSkew)) {
		return fmt.Errorf("message timestamp outside allowed skew")
	}
	sum := sha256.Sum256(e.Payload)
	if hex.EncodeToString(sum[:]) != e.PayloadDigest {
		return fmt.Errorf("payload digest mismatch")
	}
	return nil
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type EnrollmentRequest struct {
	Token         string            `json:"token"`
	PublicKeyPEM  string            `json:"public_key_pem"`
	RequestedName string            `json:"requested_name"`
	Fingerprint   DeviceFingerprint `json:"fingerprint"`
}

type EnrollmentResponse struct {
	TenantID             string `json:"tenant_id"`
	DeviceID             string `json:"device_id"`
	ClientCertificatePEM string `json:"client_certificate_pem"`
	CACertificatePEM     string `json:"ca_certificate_pem"`
	ServerEndpoint       string `json:"server_endpoint"`
}

type DeviceFingerprint struct {
	Hostname      string `json:"hostname"`
	MachineIDHash string `json:"machine_id_hash"`
	Architecture  string `json:"architecture"`
	OS            string `json:"os"`
	Kernel        string `json:"kernel"`
	Serial        string `json:"serial,omitempty"`
	Product       string `json:"product,omitempty"`
}

type Heartbeat struct {
	AgentVersion  string    `json:"agent_version"`
	Health        string    `json:"health"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	Location      *Location `json:"location,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	AccuracyM float64 `json:"accuracy_m,omitempty"`
	Source    string  `json:"source"`
	Label     string  `json:"label,omitempty"`
}

type Metric struct {
	Name         string            `json:"name"`
	Value        float64           `json:"value"`
	Unit         string            `json:"unit"`
	Labels       map[string]string `json:"labels,omitempty"`
	TimeUnixNano int64             `json:"time_unix_nano"`
}

type TelemetryBatch struct {
	Metrics []Metric `json:"metrics"`
}
