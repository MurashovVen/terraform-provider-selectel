package selectel

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/selectel/go-selvpcclient/v4/selvpcclient/resell/v2/projects"
)

func TestAccServersServerV1Basic(t *testing.T) {
	var (
		project projects.Project

		projectName = acctest.RandomWithPrefix("tf-acc")

		osName                        = "Ubuntu"
		osVersion                     = "2404"
		locationName                  = "MSK-2"
		cfgName                       = "CL25-NVMe"
		pricePlanName                 = "1 день"
		osHostName, updatedOSHostName = "hostname", "hostname1"
		osPassword, updatedOSPassword = "Passw0rd!", "Passw0rd!1"
		userScript, updatedUserScript = "#!/bin/bash", "env"
		sshKey                        = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCOIWeVNMRC7Y9jeBoG5GP3irOf/u5EbuHYixuZEmtHDtmtlnmzdcBEnpPY5OlFhjSySlUC1clCIShMXgWBfdnvk7Dbp5hgCP3Lh9pS/b8e3kxstIiGF9d7IX04DfVTOF424LlMAFbHNsrmX+uU3lizO20DljFIJNML0OdUO7eKg0XOK1OWVQlSzvZbFj39oYTSqCtoI92czQf4DdJ+0IF1/ZNewE6xPohfnZp5cl82UjYs8vxmcaiifVf7kUyQe/ilv/fZYpt59KCJBJDrTU/ko9hNxCVXrCOw7pPOQayoQ2Vir3M1AnhDMunoxFBocndgffNXVQYtA/3TXLVB7feb"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccSelectelPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckVPCV2ProjectDestroy,
		Steps: []resource.TestStep{
			// create case
			{
				Config: testAccServersServerV1(projectName, osName, osVersion, locationName, cfgName, pricePlanName, osHostName, sshKey, osPassword, userScript, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccCheckServersServerV1Exists("selectel_servers_server_v1.server_tf_acc_test_1"),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "price_plan_name", pricePlanName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "is_server_chip", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_host_name", osHostName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "user_script", userScript),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_password", osPassword),
				),
			},
			// update cases
			{
				Config: testAccServersServerV1(projectName, osName, osVersion, locationName, cfgName, pricePlanName, updatedOSHostName, sshKey, osPassword, userScript, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccCheckServersServerV1Exists("selectel_servers_server_v1.server_tf_acc_test_1"),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "price_plan_name", pricePlanName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "is_server_chip", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_host_name", updatedOSHostName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "user_script", userScript),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_password", osPassword),
				),
			},
			{
				Config: testAccServersServerV1(projectName, osName, osVersion, locationName, cfgName, pricePlanName, updatedOSHostName, sshKey, updatedOSPassword, userScript, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccCheckServersServerV1Exists("selectel_servers_server_v1.server_tf_acc_test_1"),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "price_plan_name", pricePlanName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "is_server_chip", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_host_name", updatedOSHostName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "user_script", userScript),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_password", updatedOSPassword),
				),
			},
			{
				Config: testAccServersServerV1(projectName, osName, osVersion, locationName, cfgName, pricePlanName, updatedOSHostName, sshKey, updatedOSPassword, updatedUserScript, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccCheckServersServerV1Exists("selectel_servers_server_v1.server_tf_acc_test_1"),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "price_plan_name", pricePlanName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "is_server_chip", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_host_name", updatedOSHostName),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "user_script", updatedUserScript),
					resource.TestCheckResourceAttr("selectel_servers_server_v1.server_tf_acc_test_1", "os_password", updatedOSPassword),
				),
			},
		},
	})
}

func testAccCheckServersServerV1Exists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		cl := newTestServersAPIClient(rs, testAccProvider)

		res, _, err := cl.ResourceDetails(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if res.UUID != rs.Primary.ID {
			return fmt.Errorf("resource not found %s", rs.Primary.ID)
		}

		return nil
	}
}

func testAccServersServerV1(
	projectName, osName, osVersion, locationName, cfgName, pricePlanName, osHostName, sshKey, osPassword, userScript string, isServerChip bool,
) string {
	return fmt.Sprintf(`
resource "selectel_vpc_project_v2" "project_tf_acc_test_1" {
 name        = "%s"
}

data "selectel_servers_os_v1" "os_tf_acc_test_1" {
 project_id = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"

 filter {
   name             = "%s"
   version          = "%s"
 }
}

data "selectel_servers_location_v1" "location_tf_acc_test_1" {
 project_id = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"
 filter {
   name = "%s"
 }
}

data "selectel_servers_configuration_v1" "server_configuration_tf_acc_test_1" {
 project_id     = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"
 filter {
   name           = "%s"
   is_server_chip = %t
 }
}

resource "selectel_servers_server_v1" "server_tf_acc_test_1" {
 project_id = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"

 configuration_id = "${data.selectel_servers_configuration_v1.server_configuration_tf_acc_test_1.configurations.0.id}"
 location_id      = "${data.selectel_servers_location_v1.location_tf_acc_test_1.locations[0].id}"
 os_id            = "${data.selectel_servers_os_v1.os_tf_acc_test_1.os.0.id}"
 price_plan_name  = "%s"

 os_host_name     = "%s"
 is_server_chip   = %t

 ssh_key         = "%s"

 os_password        = "%s"

 user_script = "%s"

 partitions_config {
   soft_raid_config {
     name      = "first-raid"
     level     = "raid1"
     disk_type = "SSD NVMe M.2"
   }

   disk_partitions {
     mount = "/boot"
     size  = 1
     raid  = "first-raid"
   }
   disk_partitions {
     mount = "swap"
     # size  = 12
     size_percent = 10.5
     raid         = "first-raid"
   }
   disk_partitions {
     mount = "/"
     size  = -1
     raid  = "first-raid"
   }
   disk_partitions {
     mount   = "second_folder"
     size    = 400
     raid    = "first-raid"
     fs_type = "xfs"
   }
 }
}
`, projectName, osName, osVersion, locationName, cfgName, isServerChip, pricePlanName, osHostName, isServerChip, sshKey, osPassword, userScript)
}
