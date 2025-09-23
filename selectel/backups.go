package selectel

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-selectel/selectel/internal/api/scheduledbackup"
)

func getScheduledBackupClient(d *schema.ResourceData, meta interface{}) (*scheduledbackup.ServiceClient, diag.Diagnostics) {
	config := meta.(*Config)
	projectID := d.Get("project_id").(string)

	selvpcClient, err := config.GetSelVPCClientWithProjectScope(projectID)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("can't get project-scope selvpc client for scheduled backup api: %w", err))
	}

	// todo catalog
	url := "https://ru-3.cloud.api.selcloud.ru/data-protect/v2/"

	return scheduledbackup.NewClientV2(selvpcClient.GetXAuthToken(), url), nil
}
