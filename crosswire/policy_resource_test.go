package crosswire

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPolicyResource(t *testing.T) {
	name := RandomStringGenerator(16)
	terraform_resource := fmt.Sprintf("crosswire_policy.%s", name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(terraform_resource, "owner.email_address", "user@company.com"),
					resource.TestCheckResourceAttr(terraform_resource, "name", name),
					resource.TestCheckResourceAttr(terraform_resource, "entitlements.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "entitlements.*", map[string]string{
						"provider": "CROSSWIRE", "subject": "CREATE", "object": "ENTITLEMENT"}),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "entitlements.*", map[string]string{
						"provider": "CROSSWIRE", "subject": "CREATE", "object": "PROPOSAL"}),
					resource.TestCheckResourceAttr(terraform_resource, "condition.quantifier", "ANY"),
					resource.TestCheckResourceAttr(terraform_resource, "condition.entitlements.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "condition.entitlements.*", map[string]string{
						"provider": "CROSSWIRE", "subject": "ROLE", "object": "ADMIN"}),
					resource.TestCheckResourceAttr(terraform_resource, "condition.subconditions.#", "1"),
					resource.TestCheckResourceAttr(terraform_resource, "condition.subconditions.0.quantifier", "ALL"),
					resource.TestCheckResourceAttr(terraform_resource, "condition.subconditions.0.entitlements.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "condition.subconditions.0.entitlements.*", map[string]string{
						"provider": "CROSSWIRE", "subject": "READ", "object": "POLICY"}),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "condition.subconditions.0.entitlements.*", map[string]string{
						"provider": "CROSSWIRE", "subject": "CREATE", "object": "POLICY"}),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "condition.subconditions.0.entitlements.*", map[string]string{
						"provider": "CROSSWIRE", "subject": "READ", "object": "ENTITLEMENT"}),
					resource.TestCheckResourceAttr(terraform_resource, "user_approvers.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "user_approvers.*", map[string]string{
						"email_address": "approver@company.com"}),
					resource.TestCheckResourceAttr(terraform_resource, "entitlement_approvers.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(terraform_resource, "entitlement_approvers.*", map[string]string{
						"provider": "CROSSWIRE", "subject": "APPROVE", "object": "PROPOSAL"}),
					resource.TestCheckResourceAttr(terraform_resource, "approval_behavior", "ANY"),
					resource.TestCheckResourceAttr(terraform_resource, "special_approver", "NONE"),
					resource.TestCheckResourceAttr(terraform_resource, "state", "ACTIVE"),
					resource.TestCheckNoResourceAttr(terraform_resource, "ttl"),
				),
			},
			// ImportState testing
			// Update and Read testing
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPolicyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "crosswire_policy" "%[1]s" {
  owner = {
    email_address = "user@company.com"
  }
  name = "%[1]s"
  entitlements = [
    {
      provider = "CROSSWIRE"
      subject  = "CREATE"
      object   = "ENTITLEMENT"
    },
    {
      provider = "CROSSWIRE"
      subject  = "CREATE"
      object   = "PROPOSAL"
    }
  ]
  condition = {
    quantifier = "ANY"
    entitlements = [
      {
        provider = "CROSSWIRE"
        subject  = "ROLE"
        object   = "ADMIN"
      }
    ]
    subconditions = [
      {
        quantifier : "ALL"
        entitlements : [
          {
            provider : "CROSSWIRE"
            subject : "READ"
            object : "POLICY"
          },
          {
            provider : "CROSSWIRE"
            subject : "CREATE"
            object : "POLICY"
          },
          {
            provider : "CROSSWIRE"
            subject : "READ"
            object : "ENTITLEMENT"
          }
        ]
      }
    ]
  }
  user_approvers = [
    {
      email_address = "approver@company.com"
    }
  ]
  entitlement_approvers = [
    {
      provider = "CROSSWIRE"
      subject  = "APPROVE"
      object   = "PROPOSAL"
    }
  ]
  approval_behavior = "ANY"
}
`, name)
}
