terraform {
  required_version = ">= 0.12"
  required_providers {
    akamai = {
      source = "akamai/akamai"
    }
  }
}

provider "akamai" {}

data "akamai_contract" "contract" {}

data "akamai_group" "group" {}

// NOTE: Please review the Provider Getting Started documentation before creating a Primary zone
resource "akamai_dns_zone" "test_primary_zone" {
  contract       = data.akamai_contract.contract.id
  group          = data.akamai_group.group.id
  zone           = "example_primary.net"
  type           = "PRIMARY"
  comment        = "This is a test  primary zone"
  sign_and_serve = false
}

resource "akamai_dns_zone" "test_secondary_zone" {
  contract       = data.akamai_contract.contract.id
  group          = data.akamai_group.group.id
  zone           = "example_secondary.net"
  masters        = ["1.2.3.4", "1.2.3.5"]
  type           = "secondary"
  comment        = "This is a test secondary zone"
  sign_and_serve = false
}

data "akamai_authorities_set" "ns" {
  contract = data.akamai_contract.contract.id
}
