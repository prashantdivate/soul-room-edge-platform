package audit

import (
	"encoding/json"

	"github.com/unified-fleet/cloud-infra/internal/model"
	"github.com/unified-fleet/cloud-infra/internal/storage"
	"github.com/unified-fleet/cloud-infra/internal/tenancy"
)

func Record(store *storage.Store, tc tenancy.Context, action, resourceType, resourceID, result string, summary map[string]string) {
	raw, _ := json.Marshal(summary)
	_ = store.Audit(model.AuditEvent{
		TenantID:     tc.TenantID,
		ActorID:      tc.ActorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Result:       result,
		Summary:      raw,
	})
}
