resource "dokku_plugin" "postgres" {
  name = "postgres"
  url  = "https://github.com/dokku/dokku-postgres.git"
}

# Requires plugin to be installed
# https://github.com/dokku/dokku-postgres
resource "dokku_postgres" "demo" {
  service_name = "demo"
  # hots:port or port to expose service to host machine (optional)
  expose = "7596"

  depends_on = [
    dokku_plugin.postgres
  ]
}
