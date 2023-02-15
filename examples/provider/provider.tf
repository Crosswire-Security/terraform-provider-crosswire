terraform {
  required_providers {
    crosswire = {
      source = "registry.terraform.io/crosswire-security/crosswire"
    }
  }
}

provider "crosswire" {
  api_key = "5knDBzK/1fnWmOm9mfHI0uiAIQi8IpKCrkhapJsA810="
}

resource "crosswire_policy" "resource_name" {
  owner = {
    email_address = "elton@crosswire.dev"
  }
  name = "test policy"
  entitlements = [
    {
      provider = "CROSSWIRE"
      subject = "APPROVE"
      object = "PROPOSAL"
    },
    {
      provider = "CROSSWIRE"
      subject = "CREATE"
      object = "POLICY"
    }
  ]
  condition = {
    quantifier = "ANY"
    entitlements = [
      {
        provider = "CROSSWIRE"
        subject = "ROLE"
        object = "ADMIN"
      }
    ]
    subconditions = [
      {
        quantifier: "ALL"
        entitlements: [
          {
            provider: "CROSSWIRE"
            subject: "READ"
            object: "PROPOSAL"
          },
          {
            provider: "CROSSWIRE"
            subject: "CREATE"
            object: "PROPOSAL"
          },
          {
            provider: "CROSSWIRE"
            subject: "APPROVE"
            object: "PROPOSAL"
          }
        ]
      }
    ]
  }
  user_approvers = [
    {
      email_address = "elton@crosswire.dev"
    }
  ]
  entitlement_approvers = [
    {
      provider = "CROSSWIRE"
      subject = "APPROVE"
      object = "PROPOSAL"
    }
  ]
  approval_behavior = "ANY"
}
