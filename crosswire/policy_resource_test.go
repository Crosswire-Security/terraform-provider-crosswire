package crosswire

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scaffolding_policy.test", "configurable_attribute", "one"),
					resource.TestCheckResourceAttr("scaffolding_policy.test", "id", "policy-id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "scaffolding_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// policy code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				ImportStateVerifyIgnore: []string{"configurable_attribute"},
			},
			// Update and Read testing
			{
				Config: testAccPolicyResourceConfig("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scaffolding_policy.test", "configurable_attribute", "two"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPolicyResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "scaffolding_policy" "test" {
  configurable_attribute = %[1]q
}
`, configurableAttribute)
}
