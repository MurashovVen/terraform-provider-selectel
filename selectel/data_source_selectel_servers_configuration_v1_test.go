package selectel

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/selectel/go-selvpcclient/v4/selvpcclient/resell/v2/projects"

	"github.com/terraform-providers/terraform-provider-selectel/selectel/internal/api/servers"
)

func TestAccServersConfigurationV1Basic(t *testing.T) {
	var (
		serverConfiguration *servers.Server
		project             projects.Project
	)

	projectName := acctest.RandomWithPrefix("tf-acc")
	configurationName := "EL50-SSD"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccSelectelPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckVPCV2ProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServersConfigurationV1Basic(projectName, configurationName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccServersConfigurationV1Exists("data.selectel_dbaas_datastore_type_v1.server_configuration_tf_acc_test_1", serverConfiguration, configurationName),
					resource.TestCheckResourceAttr("data.selectel_servers_configuration_v1.server_configuration_tf_acc_test_1", "configurations[0].configuration.name", configurationName),
				),
			},
		},
	})
}

func testAccServersConfigurationV1Exists(
	n string, server *servers.Server, serverName string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		ctx := context.Background()

		dsClient := newTestServersAPIClient(rs, testAccProvider)

		serversFromAPI, _, err := dsClient.Servers(ctx, false)
		if err != nil {
			return err
		}

		srvFromAPI := serversFromAPI.FindOneByName(serverName)

		if srvFromAPI == nil {
			return fmt.Errorf("server %s not found", serverName)
		}

		*server = *srvFromAPI

		return nil
	}
}

func testAccServersConfigurationV1Basic(projectName, configurationName string) string {
	return fmt.Sprintf(`
resource "selectel_vpc_project_v2" "project_tf_acc_test_1" {
  name        = "%s"
}

data "selectel_servers_configuration_v1" "server_configuration_tf_acc_test_1" {
  project_id = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"

  filter {
    name           = "%s"
    is_server_chip = false
  }
}
`, projectName, configurationName)
}

func TestAccServersConfigurationV1ServerChipBasic(t *testing.T) {
	var (
		serversConfiguration *servers.Server
		project              projects.Project
	)

	projectName := acctest.RandomWithPrefix("tf-acc")
	configurationName := "EL50-SSD"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccSelectelPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckVPCV2ProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServersConfigurationV1ServerChipBasic(projectName, configurationName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccServersConfigurationV1ServerChipExists("data.selectel_servers_configuration_v1.serverchip_configuration_tf_acc_test_1", serversConfiguration, configurationName),
					resource.TestCheckResourceAttr("data.selectel_servers_configuration_v1.serverchip_configuration_tf_acc_test_1", "configurations[0].configuration.name", configurationName),
				),
			},
		},
	})
}

func testAccServersConfigurationV1ServerChipExists(n string, server *servers.Server, serverName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		dsClient := newTestServersAPIClient(rs, testAccProvider)
		serversFromAPI, _, err := dsClient.Servers(context.Background(), true)
		if err != nil {
			return err
		}

		srvFromAPI := serversFromAPI.FindOneByName(serverName)
		if srvFromAPI == nil {
			return fmt.Errorf("server chip %s not found", serverName)
		}

		*server = *srvFromAPI

		return nil
	}
}

func testAccServersConfigurationV1ServerChipBasic(projectName, configurationName string) string {
	return fmt.Sprintf(`
resource "selectel_vpc_project_v2" "project_tf_acc_test_1" {
  name = "%s"
}

data "selectel_servers_configuration_v1" "serverchip_configuration_tf_acc_test_1" {
  project_id     = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"
  name           = "%s"
  is_server_chip = true
}
`, projectName, configurationName)
}
