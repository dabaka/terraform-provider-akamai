provider "akamai" {
  edgerc = "~/.edgerc"
}


resource "akamai_appsec_advanced_settings_evasive_path_match" "test" {
  config_id         = 43253
  enable_path_match = true
}
