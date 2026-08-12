package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/soul-room/cloud-infra/internal/auth"
	"github.com/soul-room/cloud-infra/internal/model"
	"github.com/soul-room/cloud-infra/internal/ota"
	"github.com/soul-room/cloud-infra/internal/tenancy"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrTenantMismatch = errors.New("tenant mismatch")
	ErrDuplicate      = errors.New("duplicate")
	ErrQuotaExceeded  = errors.New("quota exceeded")
)

type Store struct {
	path string
	mu   sync.Mutex
	data data
}

type data struct {
	Organizations    map[string]model.Organization     `json:"organizations"`
	Users            map[string]model.User             `json:"users"`
	Memberships      []model.Membership                `json:"memberships"`
	Sessions         map[string]auth.Session           `json:"sessions"`
	EnrollmentTokens map[string]model.EnrollmentToken  `json:"enrollment_tokens"`
	Devices          map[string]model.Device           `json:"devices"`
	Downstream       map[string]model.DownstreamDevice `json:"downstream_devices"`
	Telemetry        []model.Metric                    `json:"telemetry"`
	Inventory        map[string]json.RawMessage        `json:"inventory"`
	Jobs             map[string]model.Job              `json:"jobs"`
	Artifacts        map[string]model.Artifact         `json:"artifacts"`
	OTACampaigns     map[string]ota.Campaign           `json:"ota_campaigns"`
	Audit            []model.AuditEvent                `json:"audit"`
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	s.data.init()
	if path == "" {
		return s, nil
	}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &s.data); err != nil {
			return nil, err
		}
		s.data.init()
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (d *data) init() {
	if d.Organizations == nil {
		d.Organizations = map[string]model.Organization{}
	}
	if d.Users == nil {
		d.Users = map[string]model.User{}
	}
	if d.Sessions == nil {
		d.Sessions = map[string]auth.Session{}
	}
	if d.EnrollmentTokens == nil {
		d.EnrollmentTokens = map[string]model.EnrollmentToken{}
	}
	if d.Devices == nil {
		d.Devices = map[string]model.Device{}
	}
	if d.Downstream == nil {
		d.Downstream = map[string]model.DownstreamDevice{}
	}
	if d.Inventory == nil {
		d.Inventory = map[string]json.RawMessage{}
	}
	if d.Jobs == nil {
		d.Jobs = map[string]model.Job{}
	}
	if d.Artifacts == nil {
		d.Artifacts = map[string]model.Artifact{}
	}
	if d.OTACampaigns == nil {
		d.OTACampaigns = map[string]ota.Campaign{}
	}
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func NewID(prefix string) string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return prefix + "_" + hex.EncodeToString(b[:])
}

func (s *Store) CreateOrganization(name, slug string) (model.Organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, org := range s.data.Organizations {
		if org.Slug == slug {
			return model.Organization{}, ErrDuplicate
		}
	}
	org := model.Organization{ID: NewID("org"), Name: name, Slug: slug, CreatedAt: time.Now()}
	s.data.Organizations[org.ID] = org
	return org, s.saveLocked()
}

func (s *Store) BootstrapOrganizationOwner(name, slug, email, passwordHash string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data.Organizations) > 0 || len(s.data.Users) > 0 || len(s.data.Memberships) > 0 {
		return false, nil
	}
	now := time.Now()
	org := model.Organization{ID: NewID("org"), Name: name, Slug: slug, CreatedAt: now}
	user := model.User{ID: NewID("usr"), Email: email, PasswordHash: passwordHash, ActivatedAt: now, CreatedAt: now}
	s.data.Organizations[org.ID] = org
	s.data.Users[user.ID] = user
	s.data.Memberships = append(s.data.Memberships, model.Membership{TenantID: org.ID, UserID: user.ID, Role: "organization_owner"})
	if err := s.saveLocked(); err != nil {
		delete(s.data.Organizations, org.ID)
		delete(s.data.Users, user.ID)
		s.data.Memberships = s.data.Memberships[:0]
		return false, err
	}
	return true, nil
}

func (s *Store) ListOrganizations() []model.Organization {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Organization, 0, len(s.data.Organizations))
	for _, org := range s.data.Organizations {
		out = append(out, org)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Store) CreateUser(email, passwordHash string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Email == email {
			return model.User{}, ErrDuplicate
		}
	}
	u := model.User{ID: NewID("usr"), Email: email, PasswordHash: passwordHash, ActivatedAt: time.Now(), CreatedAt: time.Now()}
	s.data.Users[u.ID] = u
	return u, s.saveLocked()
}

func (s *Store) CreateTenantUser(tc tenancy.Context, email, passwordHash, role string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Email == email {
			return model.User{}, ErrDuplicate
		}
	}
	now := time.Now()
	u := model.User{ID: NewID("usr"), Email: email, PasswordHash: passwordHash, Role: role, ActivatedAt: now, CreatedAt: now}
	s.data.Users[u.ID] = u
	s.data.Memberships = append(s.data.Memberships, model.Membership{TenantID: tc.TenantID, UserID: u.ID, Role: role})
	if err := s.saveLocked(); err != nil {
		delete(s.data.Users, u.ID)
		s.data.Memberships = s.data.Memberships[:len(s.data.Memberships)-1]
		return model.User{}, err
	}
	u.PasswordHash = ""
	return u, nil
}

func (s *Store) UpsertUserPassword(email, passwordHash string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, u := range s.data.Users {
		if u.Email == email {
			u.PasswordHash = passwordHash
			u.ActivatedAt = time.Now()
			u.LockedUntil = time.Time{}
			u.FailedLoginCount = 0
			s.data.Users[id] = u
			return u, s.saveLocked()
		}
	}
	u := model.User{ID: NewID("usr"), Email: email, PasswordHash: passwordHash, ActivatedAt: time.Now(), CreatedAt: time.Now()}
	s.data.Users[u.ID] = u
	return u, s.saveLocked()
}

func (s *Store) FindUserByEmail(email string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Email == email {
			return u, nil
		}
	}
	return model.User{}, ErrNotFound
}

func (s *Store) AddMembership(m model.Membership) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Memberships {
		if existing.TenantID == m.TenantID && existing.UserID == m.UserID {
			return ErrDuplicate
		}
	}
	s.data.Memberships = append(s.data.Memberships, m)
	return s.saveLocked()
}

func (s *Store) Memberships(userID string) []model.Membership {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Membership
	for _, m := range s.data.Memberships {
		if m.UserID == userID {
			out = append(out, m)
		}
	}
	return out
}

func (s *Store) ListUsers(tc tenancy.Context) []model.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	roles := map[string]string{}
	for _, membership := range s.data.Memberships {
		if membership.TenantID == tc.TenantID {
			roles[membership.UserID] = membership.Role
		}
	}
	out := make([]model.User, 0, len(roles))
	for id, user := range s.data.Users {
		if role, allowed := roles[id]; allowed {
			user.PasswordHash = ""
			user.Role = role
			out = append(out, user)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out
}

func (s *Store) SaveSession(session auth.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Sessions[session.TokenHash] = session
	return s.saveLocked()
}

func (s *Store) SessionByToken(token string) (auth.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.data.Sessions[auth.HashToken(token)]
	if !ok || !session.RevokedAt.IsZero() || time.Now().After(session.ExpiresAt) {
		return auth.Session{}, ErrNotFound
	}
	return session, nil
}

func (s *Store) CreateEnrollmentToken(tc tenancy.Context, token model.EnrollmentToken) (model.EnrollmentToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token.ID = NewID("enr")
	token.TenantID = tc.TenantID
	token.CreatedBy = tc.ActorID
	token.CreatedAt = time.Now()
	if token.MaxDevices == 0 {
		token.MaxDevices = 1
	}
	s.data.EnrollmentTokens[token.ID] = token
	return token, s.saveLocked()
}

func (s *Store) ListEnrollmentTokens(tc tenancy.Context) []model.EnrollmentToken {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []model.EnrollmentToken{}
	for _, token := range s.data.EnrollmentTokens {
		if token.TenantID == tc.TenantID {
			token.TokenHash = ""
			token.PlainToken = ""
			out = append(out, token)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) ConsumeEnrollmentToken(tokenHash, serial, product string) (model.EnrollmentToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, token := range s.data.EnrollmentTokens {
		if token.TokenHash != tokenHash {
			continue
		}
		if !token.RevokedAt.IsZero() || time.Now().After(token.ExpiresAt) || token.UsedCount >= token.MaxDevices {
			return model.EnrollmentToken{}, ErrNotFound
		}
		if token.ExpectedSerial != "" && token.ExpectedSerial != serial {
			return model.EnrollmentToken{}, ErrTenantMismatch
		}
		if token.ExpectedProduct != "" && token.ExpectedProduct != product {
			return model.EnrollmentToken{}, ErrTenantMismatch
		}
		token.UsedCount++
		s.data.EnrollmentTokens[id] = token
		return token, s.saveLocked()
	}
	return model.EnrollmentToken{}, ErrNotFound
}

func (s *Store) CreateDevice(device model.Device) (model.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	device.ID = NewID("dev")
	device.CreatedAt = time.Now()
	if device.Presence == "" {
		device.Presence = "offline"
	}
	if device.EnrollmentStatus == "" {
		device.EnrollmentStatus = "enrolled"
	}
	s.data.Devices[device.ID] = device
	return device, s.saveLocked()
}

func (s *Store) ListDevices(tc tenancy.Context) []model.Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Device
	for _, d := range s.data.Devices {
		if d.TenantID == tc.TenantID {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	return out
}

func (s *Store) ListDownstream(tc tenancy.Context) []model.DownstreamDevice {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []model.DownstreamDevice{}
	for _, device := range s.data.Downstream {
		if device.TenantID == tc.TenantID {
			out = append(out, device)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProtocolAddress < out[j].ProtocolAddress })
	return out
}

func (s *Store) UpsertDownstream(device model.DownstreamDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if device.ID == "" {
		device.ID = NewID("child")
	}
	s.data.Downstream[device.ID] = device
	return s.saveLocked()
}

func (s *Store) GetDevice(tc tenancy.Context, id string) (model.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.data.Devices[id]
	if !ok {
		return model.Device{}, ErrNotFound
	}
	if d.TenantID != tc.TenantID {
		return model.Device{}, ErrTenantMismatch
	}
	return d, nil
}

func (s *Store) UpdateDevice(device model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.data.Devices[device.ID]
	if !ok {
		return ErrNotFound
	}
	if existing.TenantID != device.TenantID {
		return ErrTenantMismatch
	}
	s.data.Devices[device.ID] = device
	return s.saveLocked()
}

func (s *Store) DeleteDevice(tc tenancy.Context, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	device, ok := s.data.Devices[deviceID]
	if !ok {
		return ErrNotFound
	}
	if device.TenantID != tc.TenantID {
		return ErrTenantMismatch
	}
	delete(s.data.Devices, deviceID)
	delete(s.data.Inventory, deviceID)
	for id, downstream := range s.data.Downstream {
		if downstream.ParentGatewayID == deviceID {
			delete(s.data.Downstream, id)
		}
	}
	telemetry := s.data.Telemetry[:0]
	for _, metric := range s.data.Telemetry {
		if metric.DeviceID != deviceID {
			telemetry = append(telemetry, metric)
		}
	}
	s.data.Telemetry = telemetry
	for id, job := range s.data.Jobs {
		if job.DeviceID == deviceID {
			delete(s.data.Jobs, id)
		}
	}
	for id, campaign := range s.data.OTACampaigns {
		if campaign.TenantID != tc.TenantID {
			continue
		}
		targets := campaign.TargetIDs[:0]
		for _, targetID := range campaign.TargetIDs {
			if targetID != deviceID {
				targets = append(targets, targetID)
			}
		}
		if len(targets) == 0 {
			delete(s.data.OTACampaigns, id)
		} else {
			campaign.TargetIDs = targets
			s.data.OTACampaigns[id] = campaign
		}
	}
	return s.saveLocked()
}

func (s *Store) AddTelemetry(tc tenancy.Context, metrics []model.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(metrics) > 5000 {
		return ErrQuotaExceeded
	}
	for _, m := range metrics {
		d, ok := s.data.Devices[m.DeviceID]
		if !ok || d.TenantID != tc.TenantID {
			return ErrTenantMismatch
		}
		if len(m.Labels) > 12 {
			return ErrQuotaExceeded
		}
		m.ReceivedAt = time.Now()
		s.data.Telemetry = append(s.data.Telemetry, m)
	}
	return s.saveLocked()
}

func (s *Store) ListTelemetry(tc tenancy.Context, deviceID string) []model.Metric {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Metric
	for _, m := range s.data.Telemetry {
		if m.DeviceID != deviceID {
			continue
		}
		d, ok := s.data.Devices[m.DeviceID]
		if ok && d.TenantID == tc.TenantID {
			out = append(out, m)
		}
	}
	return out
}

func (s *Store) SaveInventory(tc tenancy.Context, deviceID string, payload json.RawMessage) error {
	if _, err := s.GetDevice(tc, deviceID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Inventory[deviceID] = append(json.RawMessage(nil), payload...)
	return s.saveLocked()
}

func (s *Store) ListInventory(tc tenancy.Context) map[string]json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]json.RawMessage{}
	for deviceID, inventory := range s.data.Inventory {
		device, ok := s.data.Devices[deviceID]
		if ok && device.TenantID == tc.TenantID {
			out[deviceID] = append(json.RawMessage(nil), inventory...)
		}
	}
	return out
}

func (s *Store) CreateJob(tc tenancy.Context, job model.Job) (model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.data.Devices[job.DeviceID]
	if !ok || d.TenantID != tc.TenantID {
		return model.Job{}, ErrTenantMismatch
	}
	for _, existing := range s.data.Jobs {
		if job.IdempotencyKey != "" && existing.TenantID == tc.TenantID && existing.IdempotencyKey == job.IdempotencyKey {
			return existing, nil
		}
	}
	job.ID = NewID("job")
	job.TenantID = tc.TenantID
	job.CreatedAt = time.Now()
	if job.State == "" {
		job.State = "queued"
	}
	if job.ApprovalStatus == "" {
		job.ApprovalStatus = "approved"
	}
	s.data.Jobs[job.ID] = job
	return job, s.saveLocked()
}

func (s *Store) ListJobs(tc tenancy.Context) []model.Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []model.Job{}
	for _, job := range s.data.Jobs {
		if job.TenantID == tc.TenantID {
			out = append(out, job)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) DeleteJob(tc tenancy.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.data.Jobs[jobID]
	if !ok {
		return ErrNotFound
	}
	if job.TenantID != tc.TenantID {
		return ErrTenantMismatch
	}
	if job.State != "succeeded" && job.State != "failed" && job.State != "expired" && job.State != "cancelled" {
		return errors.New("only terminal jobs can be removed from history")
	}
	delete(s.data.Jobs, jobID)
	return s.saveLocked()
}

func (s *Store) UpsertArtifact(artifact model.Artifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if artifact.ID == "" {
		artifact.ID = NewID("art")
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now()
	}
	s.data.Artifacts[artifact.ID] = artifact
	return s.saveLocked()
}

func (s *Store) ListArtifacts(tc tenancy.Context) []model.Artifact {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []model.Artifact{}
	for _, artifact := range s.data.Artifacts {
		if artifact.TenantID == tc.TenantID {
			out = append(out, artifact)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) CreateOTACampaign(tc tenancy.Context, campaign ota.Campaign) (ota.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if campaign.TenantID != "" && campaign.TenantID != tc.TenantID {
		return ota.Campaign{}, ErrTenantMismatch
	}
	campaign.ID = NewID("ota")
	campaign.TenantID = tc.TenantID
	campaign.CreatedAt = time.Now()
	if campaign.State == "" {
		campaign.State = "ready"
	}
	s.data.OTACampaigns[campaign.ID] = campaign
	return campaign, s.saveLocked()
}

func (s *Store) ListOTACampaigns(tc tenancy.Context) []ota.Campaign {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ota.Campaign{}
	for _, campaign := range s.data.OTACampaigns {
		if campaign.TenantID == tc.TenantID {
			out = append(out, campaign)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) GetOTACampaign(tc tenancy.Context, campaignID string) (ota.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	campaign, ok := s.data.OTACampaigns[campaignID]
	if !ok {
		return ota.Campaign{}, ErrNotFound
	}
	if campaign.TenantID != tc.TenantID {
		return ota.Campaign{}, ErrTenantMismatch
	}
	return campaign, nil
}

func (s *Store) UpdateOTACampaign(tc tenancy.Context, campaign ota.Campaign) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.data.OTACampaigns[campaign.ID]
	if !ok {
		return ErrNotFound
	}
	if existing.TenantID != tc.TenantID || campaign.TenantID != tc.TenantID {
		return ErrTenantMismatch
	}
	s.data.OTACampaigns[campaign.ID] = campaign
	return s.saveLocked()
}

func (s *Store) JobsForDeployment(tc tenancy.Context, deploymentID string) []model.Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []model.Job{}
	for _, job := range s.data.Jobs {
		if job.TenantID == tc.TenantID && job.DeploymentID == deploymentID {
			out = append(out, job)
		}
	}
	return out
}

func (s *Store) NextJobForDevice(tc tenancy.Context, deviceID string) (model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, job := range s.data.Jobs {
		if job.TenantID == tc.TenantID && job.DeviceID == deviceID && (job.State == "queued" || job.State == "delivered") {
			if now.After(job.ExpiresAt) {
				job.State = "expired"
				s.data.Jobs[id] = job
				continue
			}
			job.State = "delivered"
			job.Attempt++
			s.data.Jobs[id] = job
			_ = s.saveLocked()
			return job, nil
		}
	}
	return model.Job{}, ErrNotFound
}

func (s *Store) CompleteJob(tc tenancy.Context, jobID string, result json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.data.Jobs[jobID]
	if !ok {
		return ErrNotFound
	}
	if job.TenantID != tc.TenantID {
		return ErrTenantMismatch
	}
	var reported struct {
		State string `json:"state"`
	}
	_ = json.Unmarshal(result, &reported)
	job.State = reported.State
	if job.State != "succeeded" && job.State != "failed" {
		job.State = "failed"
	}
	job.Result = result
	s.data.Jobs[jobID] = job
	if campaign, ok := s.data.OTACampaigns[job.DeploymentID]; ok && campaign.TenantID == tc.TenantID {
		campaign.State = campaignState(campaign, s.data.Jobs)
		s.data.OTACampaigns[campaign.ID] = campaign
	}
	return s.saveLocked()
}

func campaignState(campaign ota.Campaign, allJobs map[string]model.Job) string {
	jobs := []model.Job{}
	for _, job := range allJobs {
		if job.TenantID == campaign.TenantID && job.DeploymentID == campaign.ID {
			jobs = append(jobs, job)
		}
	}
	if len(jobs) == 0 {
		return campaign.State
	}
	allSucceeded := true
	for _, job := range jobs {
		if job.State == "failed" || job.State == "expired" {
			return "failed"
		}
		if job.State != "succeeded" {
			allSucceeded = false
		}
	}
	if allSucceeded {
		if len(jobs) < len(campaign.TargetIDs) {
			return "canary_complete"
		}
		return "completed"
	}
	return "running"
}

func (s *Store) Audit(event model.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event.ID = NewID("aud")
	event.CreatedAt = time.Now()
	s.data.Audit = append(s.data.Audit, event)
	return s.saveLocked()
}

func (s *Store) ListAudit(tc tenancy.Context) []model.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.AuditEvent
	for _, e := range s.data.Audit {
		if e.TenantID == tc.TenantID {
			out = append(out, e)
		}
	}
	return out
}
