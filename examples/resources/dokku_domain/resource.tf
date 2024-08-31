# For global domains setup
# For app-specific domains use app_resource.domains attribute
# https://dokku.com/docs/configuration/domains/
resource "dokku_domain" "demo" {
  domain = "example.com"
}
