package profiles

type Profile struct {
	ID                   string   `json:"id"`
	TenantID             string   `json:"tenant_id"`
	Name                 string   `json:"name"`
	AllowedJobTypes      []string `json:"allowed_job_types"`
	UpdateChannel        string   `json:"update_channel"`
	DeploymentRing       string   `json:"deployment_ring"`
	ConfigurationVersion string   `json:"configuration_version"`
}
