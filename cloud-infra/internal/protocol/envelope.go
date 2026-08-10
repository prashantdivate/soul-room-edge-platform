package protocol

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

const Version = "fleet.v1"

type Envelope struct {
	ProtocolVersion       string          `json:"protocol_version"`
	MessageID             string          `json:"message_id"`
	CorrelationID         string          `json:"correlation_id,omitempty"`
	TenantID              string          `json:"tenant_id,omitempty"`
	DeviceID              string          `json:"device_id,omitempty"`
	GatewayID             string          `json:"gateway_id,omitempty"`
	Sequence              uint64          `json:"sequence,omitempty"`
	TimestampUnixNano     int64           `json:"timestamp_unix_nano"`
	LegacyCreatedUnixNano int64           `json:"created_unix_nano,omitempty"`
	PayloadSchemaVersion  string          `json:"payload_schema_version"`
	PayloadDigest         string          `json:"payload_digest"`
	Payload               json.RawMessage `json:"payload"`
}

func Wrap(tenantID, deviceID, schema string, sequence uint64, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	sum := sha256.Sum256(raw)
	return Envelope{
		ProtocolVersion:      Version,
		MessageID:            NewMessageID(),
		TenantID:             tenantID,
		DeviceID:             deviceID,
		Sequence:             sequence,
		TimestampUnixNano:    time.Now().UnixNano(),
		PayloadSchemaVersion: schema,
		PayloadDigest:        hex.EncodeToString(sum[:]),
		Payload:              raw,
	}, nil
}

func (e Envelope) Validate(now time.Time, skew time.Duration) error {
	if e.ProtocolVersion != Version && e.ProtocolVersion != "edge.v1" {
		return errors.New("unsupported protocol version")
	}
	if e.MessageID == "" {
		return errors.New("missing message_id")
	}
	ts := time.Unix(0, e.TimestampUnixNano)
	if e.TimestampUnixNano == 0 {
		ts = time.Unix(0, e.LegacyCreatedUnixNano)
	}
	if ts.After(now.Add(skew)) || ts.Before(now.Add(-skew)) {
		return errors.New("timestamp outside allowed skew")
	}
	sum := sha256.Sum256(e.Payload)
	if e.PayloadDigest != "" && hex.EncodeToString(sum[:]) != e.PayloadDigest {
		return errors.New("payload digest mismatch")
	}
	return nil
}

func NewMessageID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type EnrollmentRequest struct {
	Token           string            `json:"token"`
	EnrollmentToken string            `json:"enrollment_token,omitempty"`
	PublicKeyPEM    string            `json:"public_key_pem"`
	RequestedName   string            `json:"requested_name"`
	Fingerprint     DeviceFingerprint `json:"fingerprint"`
	Inventory       json.RawMessage   `json:"inventory,omitempty"`
}

type DeviceFingerprint struct {
	Hostname      string `json:"hostname"`
	MachineIDHash string `json:"machine_id_hash"`
	Serial        string `json:"serial,omitempty"`
	Product       string `json:"product,omitempty"`
	Architecture  string `json:"architecture"`
	OS            string `json:"os"`
	Kernel        string `json:"kernel"`
}

type EnrollmentResponse struct {
	TenantID             string `json:"tenant_id"`
	DeviceID             string `json:"device_id"`
	ClientCertificatePEM string `json:"client_certificate_pem"`
	CACertificatePEM     string `json:"ca_certificate_pem"`
	ServerEndpoint       string `json:"server_endpoint"`
}

type Heartbeat struct {
	AgentVersion       string    `json:"agent_version"`
	Health             string    `json:"health"`
	OSUptimeSeconds    int64     `json:"os_uptime_seconds,omitempty"`
	AgentUptimeSeconds int64     `json:"agent_uptime_seconds,omitempty"`
	UptimeSeconds      int64     `json:"uptime_seconds,omitempty"`
	Location           *Location `json:"location,omitempty"`
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
