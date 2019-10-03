package vmc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"gitlab.eng.vmware.com/het/vmware-vmc-sdk/vapi/bindings/vmc/orgs/sddcs"
	"gitlab.eng.vmware.com/het/vmware-vmc-sdk/vapi/runtime/protocol/client"
	"os"
	"testing"
)

func TestAccResourceVmcSddc_basic(t *testing.T) {
	sddcName := "srege_test_sddc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckVmcSddcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVmcSddcConfigBasic(sddcName),
				Check: resource.ComposeTestCheckFunc(
					testCheckVmcSddcExists("vmc_sddc.sddc_1"),
				),
			},
		},
	})
}

func testCheckVmcSddcExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		sddcID := rs.Primary.Attributes["id"]
		sddcName := rs.Primary.Attributes["sddc_name"]
		orgID := rs.Primary.Attributes["org_id"]
		connector := testAccProvider.Meta().(client.Connector)
		sddcClient := sddcs.NewSddcsClientImpl(connector)

		sddc, err := sddcClient.Get(orgID, sddcID)
		if err != nil {
			return fmt.Errorf("Bad: Get on sddcApi: %s", err)
		}

		if sddc.Id != sddcID {
			return fmt.Errorf("Bad: Sddc %q does not exist", sddcName)
		}

		fmt.Printf("SDDC %s created successfully with id %s ", sddcName, sddcID)
		return nil
	}
}

func testCheckVmcSddcDestroy(s *terraform.State) error {

	connector := testAccProvider.Meta().(client.Connector)
	sddcClient := sddcs.NewSddcsClientImpl(connector)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vmc_sddc" {
			continue
		}

		sddcID := rs.Primary.Attributes["id"]
		orgID := rs.Primary.Attributes["org_id"]
		task, err := sddcClient.Delete(orgID, sddcID, nil, nil, nil)
		if err != nil {
			return fmt.Errorf("Error while deleting sddc %s, %s", sddcID, err)
		}
		err = WaitForTask(connector, orgID, task.Id)
		if err != nil {
			return fmt.Errorf("Error while waiting for task %q: %v", task.Id, err)
		}
	}

	return nil
}

func testAccVmcSddcConfigBasic(sddcName string) string {
	return fmt.Sprintf(`
provider "vmc" {
	refresh_token = %q
	
	# refresh_token = "ac5140ea-1749-4355-a892-56cff4893be0"
	# vmc_url       = "https://stg.skyscraper.vmware.com/vmc/api"
	# csp_url       = "https://console-stg.cloud.vmware.com"
}
	
data "vmc_org" "my_org" {
	id = "54937bce-8119-4fae-84f5-e5e066ee90e6"
}

data "vmc_connected_accounts" "accounts" {
	org_id = "${data.vmc_org.my_org.id}"
}

resource "vmc_sddc" "sddc_1" {
	org_id = "${data.vmc_org.my_org.id}"

	# storage_capacity    = 100
	sddc_name = %q

	vpc_cidr      = "10.2.0.0/16"
	num_host      = 1
	provider_type = "ZEROCLOUD"

	region = "US_EAST_1"

	vxlan_subnet = "192.168.1.0/24"

	delay_account_link  = false
	skip_creating_vxlan = false
	sso_domain          = "vmc.local"

	sddc_template_id    = ""
	deployment_type = "SingleAZ"

	# TODO raise exception here need to debug
	#account_link_sddc_config = [
	#	{
	#	  customer_subnet_ids  = ["subnet-13a0c249"]
	#	  connected_account_id = "${data.vmc_connected_accounts.accounts.ids.0}"
	#	},
	#  ]
}
`,
		os.Getenv("REFRESH_TOKEN"),
		sddcName,
	)
}
