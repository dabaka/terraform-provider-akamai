provider "akamai" {
  edgerc = "~/.edgerc"
}

data "akamai_appsec_advanced_settings_logging" "logging" {
  config_id = 43253
    version = 7
    security_policy_id = "AAAA_81230"
}

