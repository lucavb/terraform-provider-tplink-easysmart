package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	tfprotov6 "github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	providerpkg "github.com/lucavb/terraform-provider-tplink-easysmart/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tplinkeasysmart": providerserver.NewProtocol6WithError(providerpkg.New("test")()),
}

func TestAccSystemInfoDataSource_smoke(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	if host == "" || username == "" || password == "" {
		t.Skip("set TPLINK_EASYSMART_HOST, TPLINK_EASYSMART_USERNAME, and TPLINK_EASYSMART_PASSWORD to run acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + `
data "tplinkeasysmart_system_info" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_system_info.test", "description"),
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_system_info.test", "firmware"),
				),
			},
		},
	})
}

func TestAccPortsDataSource_smoke(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	if host == "" || username == "" || password == "" {
		t.Skip("set TPLINK_EASYSMART_HOST, TPLINK_EASYSMART_USERNAME, and TPLINK_EASYSMART_PASSWORD to run acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + `
data "tplinkeasysmart_ports" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_ports.test", "ports.0.id"),
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_ports.test", "ports.0.enabled"),
				),
			},
		},
	})
}

func TestAccVLANsDataSource_smoke(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	if host == "" || username == "" || password == "" {
		t.Skip("set TPLINK_EASYSMART_HOST, TPLINK_EASYSMART_USERNAME, and TPLINK_EASYSMART_PASSWORD to run acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + `
data "tplinkeasysmart_vlans" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_vlans.test", "enabled"),
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_vlans.test", "port_num"),
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_vlans.test", "vlan_count"),
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_vlans.test", "max_vlans"),
				),
			},
		},
	})
}

func TestAccPortPVIDsDataSource_smoke(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	if host == "" || username == "" || password == "" {
		t.Skip("set TPLINK_EASYSMART_HOST, TPLINK_EASYSMART_USERNAME, and TPLINK_EASYSMART_PASSWORD to run acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + `
data "tplinkeasysmart_port_pvids" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_port_pvids.test", "pvids.0.port_id"),
					resource.TestCheckResourceAttrSet("data.tplinkeasysmart_port_pvids.test", "pvids.0.pvid"),
				),
			},
		},
	})
}

func TestAccVLAN8021QResource_basic(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	vlanID := os.Getenv("TPLINK_EASYSMART_TEST_VLAN_ID")
	untaggedPort := os.Getenv("TPLINK_EASYSMART_TEST_VLAN_UNTAGGED_PORT")
	taggedPort := os.Getenv("TPLINK_EASYSMART_TEST_VLAN_TAGGED_PORT")
	if host == "" || username == "" || password == "" || vlanID == "" || untaggedPort == "" || taggedPort == "" {
		t.Skip("set TPLINK_EASYSMART_TEST_VLAN_ID, TPLINK_EASYSMART_TEST_VLAN_UNTAGGED_PORT, and TPLINK_EASYSMART_TEST_VLAN_TAGGED_PORT to run VLAN write acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_vlan_8021q" "test" {
  vlan_id        = %s
  name           = "TfAccVlan"
  tagged_ports   = [%s]
  untagged_ports = [%s]
}
`, vlanID, taggedPort, untaggedPort),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_vlan_8021q.test", "vlan_id", vlanID),
					resource.TestCheckResourceAttr("tplinkeasysmart_vlan_8021q.test", "name", "TfAccVlan"),
				),
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_vlan_8021q" "test" {
  vlan_id        = %s
  name           = "TfAccVlan2"
  tagged_ports   = [%s]
  untagged_ports = [%s]
}
`, vlanID, untaggedPort, taggedPort),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_vlan_8021q.test", "name", "TfAccVlan2"),
				),
			},
			{
				ResourceName:      "tplinkeasysmart_vlan_8021q.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_vlan_8021q" "test" {
  vlan_id        = %s
  name           = "TfAccVlan2"
  tagged_ports   = [%s]
  untagged_ports = [%s]
}
`, vlanID, untaggedPort, taggedPort),
				PlanOnly: true,
			},
		},
	})
}

func TestAccPortPVIDResource_basic(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	vlanID := os.Getenv("TPLINK_EASYSMART_TEST_VLAN_ID")
	portID := os.Getenv("TPLINK_EASYSMART_TEST_PVID_PORT")
	if host == "" || username == "" || password == "" || vlanID == "" || portID == "" {
		t.Skip("set TPLINK_EASYSMART_TEST_VLAN_ID and TPLINK_EASYSMART_TEST_PVID_PORT to run PVID write acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_vlan_8021q" "test" {
  vlan_id        = %s
  name           = "TfAccPvid"
  tagged_ports   = []
  untagged_ports = [%s]
}

resource "tplinkeasysmart_port_pvid" "test" {
  port_id = %s
  pvid    = %s
}
`, vlanID, portID, portID, vlanID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_port_pvid.test", "port_id", portID),
					resource.TestCheckResourceAttr("tplinkeasysmart_port_pvid.test", "pvid", vlanID),
				),
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_vlan_8021q" "test" {
  vlan_id        = %s
  name           = "TfAccPvid"
  tagged_ports   = []
  untagged_ports = [%s]
}

resource "tplinkeasysmart_port_pvid" "test" {
  port_id = %s
  pvid    = 1
}
`, vlanID, portID, portID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_port_pvid.test", "pvid", "1"),
				),
			},
			{
				ResourceName:      "tplinkeasysmart_port_pvid.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_vlan_8021q" "test" {
  vlan_id        = %s
  name           = "TfAccPvid"
  tagged_ports   = []
  untagged_ports = [%s]
}

resource "tplinkeasysmart_port_pvid" "test" {
  port_id = %s
  pvid    = 1
}
`, vlanID, portID, portID),
				PlanOnly: true,
			},
		},
	})
}

func TestAccPortSettingResource_basic(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	portID := os.Getenv("TPLINK_EASYSMART_TEST_PORT_SETTING_PORT")
	speed := os.Getenv("TPLINK_EASYSMART_TEST_PORT_SETTING_SPEED")
	enabled := os.Getenv("TPLINK_EASYSMART_TEST_PORT_SETTING_ENABLED")
	if host == "" || username == "" || password == "" || portID == "" || speed == "" || enabled == "" {
		t.Skip("set TPLINK_EASYSMART_TEST_PORT_SETTING_PORT, TPLINK_EASYSMART_TEST_PORT_SETTING_SPEED, and TPLINK_EASYSMART_TEST_PORT_SETTING_ENABLED to run port-setting acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_port_setting" "test" {
  port_id              = %s
  enabled              = %s
  speed_config         = %s
  flow_control_config  = 1
}
`, portID, enabled, speed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_port_setting.test", "port_id", portID),
					resource.TestCheckResourceAttr("tplinkeasysmart_port_setting.test", "flow_control_config", "1"),
				),
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_port_setting" "test" {
  port_id              = %s
  enabled              = %s
  speed_config         = %s
  flow_control_config  = 0
}
`, portID, enabled, speed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_port_setting.test", "flow_control_config", "0"),
				),
			},
		},
	})
}

func TestAccIGMPSnoopingResource_basic(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	enabled := os.Getenv("TPLINK_EASYSMART_TEST_IGMP_ENABLED")
	suppressionPrimary := os.Getenv("TPLINK_EASYSMART_TEST_IGMP_REPORT_SUPPRESSION_PRIMARY")
	suppressionSecondary := os.Getenv("TPLINK_EASYSMART_TEST_IGMP_REPORT_SUPPRESSION_SECONDARY")
	if host == "" || username == "" || password == "" || enabled == "" || suppressionPrimary == "" || suppressionSecondary == "" {
		t.Skip("set TPLINK_EASYSMART_TEST_IGMP_ENABLED, TPLINK_EASYSMART_TEST_IGMP_REPORT_SUPPRESSION_PRIMARY, and TPLINK_EASYSMART_TEST_IGMP_REPORT_SUPPRESSION_SECONDARY to run IGMP snooping acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_igmp_snooping" "test" {
  enabled                    = %s
  report_message_suppression = %s
}
`, enabled, suppressionPrimary),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_igmp_snooping.test", "enabled", enabled),
					resource.TestCheckResourceAttr("tplinkeasysmart_igmp_snooping.test", "report_message_suppression", suppressionPrimary),
				),
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_igmp_snooping" "test" {
  enabled                    = %s
  report_message_suppression = %s
}
`, enabled, suppressionSecondary),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_igmp_snooping.test", "report_message_suppression", suppressionSecondary),
				),
			},
			{
				ResourceName:      "tplinkeasysmart_igmp_snooping.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "igmp_snooping",
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_igmp_snooping" "test" {
  enabled                    = %s
  report_message_suppression = %s
}
`, enabled, suppressionSecondary),
				PlanOnly: true,
			},
		},
	})
}

func TestAccLAGResource_basic(t *testing.T) {
	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	groupID := os.Getenv("TPLINK_EASYSMART_TEST_LAG_GROUP")
	portsPrimary := os.Getenv("TPLINK_EASYSMART_TEST_LAG_PORTS_PRIMARY")
	portsSecondary := os.Getenv("TPLINK_EASYSMART_TEST_LAG_PORTS_SECONDARY")
	if host == "" || username == "" || password == "" || groupID == "" || portsPrimary == "" || portsSecondary == "" {
		t.Skip("set TPLINK_EASYSMART_TEST_LAG_GROUP, TPLINK_EASYSMART_TEST_LAG_PORTS_PRIMARY, and TPLINK_EASYSMART_TEST_LAG_PORTS_SECONDARY to run LAG acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_lag" "test" {
  group_id = %s
  ports    = [%s]
}
`, groupID, portsPrimary),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_lag.test", "group_id", groupID),
					resource.TestCheckResourceAttrSet("tplinkeasysmart_lag.test", "id"),
				),
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_lag" "test" {
  group_id = %s
  ports    = [%s]
}
`, groupID, portsSecondary),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tplinkeasysmart_lag.test", "group_id", groupID),
				),
			},
			{
				ResourceName:      "tplinkeasysmart_lag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig(host, username, password) + fmt.Sprintf(`
resource "tplinkeasysmart_lag" "test" {
  group_id = %s
  ports    = [%s]
}
`, groupID, portsSecondary),
				PlanOnly: true,
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	_, _, _ = testAccProviderEnv(t)
}

func testAccProviderEnv(t *testing.T) (string, string, string) {
	t.Helper()

	host := os.Getenv("TPLINK_EASYSMART_HOST")
	username := os.Getenv("TPLINK_EASYSMART_USERNAME")
	password := os.Getenv("TPLINK_EASYSMART_PASSWORD")
	for key, value := range map[string]string{
		"TPLINK_EASYSMART_HOST":     host,
		"TPLINK_EASYSMART_USERNAME": username,
		"TPLINK_EASYSMART_PASSWORD": password,
	} {
		if value == "" {
			t.Fatalf("%s must be set for acceptance tests", key)
		}
	}
	return host, username, password
}

func testAccProviderConfig(host string, username string, password string) string {
	return fmt.Sprintf(`
provider "tplinkeasysmart" {
  host          = %q
  username      = %q
  password      = %q
  insecure_http = true
}
`, host, username, password)
}
