package selectel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/selectel/go-selvpcclient/v4/selvpcclient/resell/v2/servers"
	"github.com/stretchr/testify/assert"
	serverslocal "github.com/terraform-providers/terraform-provider-selectel/selectel/internal/api/servers"
)

func TestServersMapsFromStructs(t *testing.T) {
	serverStructs := []servers.Server{
		{
			ID:     "a208023f-69fe-4a9e-8285-dd44e94a854a",
			Name:   "fake",
			Status: "ACTIVE",
		},
	}
	expectedServersMaps := []map[string]interface{}{
		{
			"id":     "a208023f-69fe-4a9e-8285-dd44e94a854a",
			"name":   "fake",
			"status": "ACTIVE",
		},
	}

	actualServersMaps := serversMapsFromStructs(serverStructs)

	assert.Equal(t, expectedServersMaps, actualServersMaps)
}

func newTestServersAPIClient(rs *terraform.ResourceState, testAccProvider *schema.Provider) *serverslocal.ServiceClient {
	config := testAccProvider.Meta().(*Config)

	var projectID string

	if id, ok := rs.Primary.Attributes["project_id"]; ok {
		projectID = id
	}

	selvpcClient, err := config.GetSelVPCClientWithProjectScope(projectID)
	if err != nil {
		panic("can't get selvpc client for dedicated servers acc tests: " + err.Error())
	}

	url := "https://api.selectel.ru/servers/v2"

	return serverslocal.NewClientV2(selvpcClient.GetXAuthToken(), url)
}
