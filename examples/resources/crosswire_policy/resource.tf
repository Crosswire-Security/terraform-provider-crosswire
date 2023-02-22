resource "crosswire_policy" "resource_name" {
  owner = {
    email_address = "user@crosswire.io"
  }
  name = "test policy"
  entitlements = [
    {
      provider = "CROSSWIRE"
      subject  = "SUBJECT"
      object   = "OBJECT1"
    },
    {
      provider = "CROSSWIRE"
      subject  = "SUBJECT"
      object   = "OBJECT2"
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
            subject : "SUBJECT"
            object : "OBJECT3"
          },
          {
            provider : "CROSSWIRE"
            subject : "SUBJECT"
            object : "OBJECT4"
          }
        ]
      }
    ]
  }
  user_approvers = [
    {
      email_address = "approver@crosswire.io"
    }
  ]
  entitlement_approvers = [
    {
      provider = "CROSSWIRE"
      subject  = "GROUP"
      object   = "USERS"
    }
  ]
  approval_behavior = "ANY"
}
