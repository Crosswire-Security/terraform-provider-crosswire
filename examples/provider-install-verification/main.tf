terraform {
  required_providers {
    crosswire = {
      source = "registry.terraform.io/crosswire-security/crosswire"
    }
  }
}

provider "crosswire" {
  host = "127.0.0.1:8080"
}

