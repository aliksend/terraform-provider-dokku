resource "dokku_plugin" "postgres" {
  name = "postgres"
  url  = "https://github.com/dokku/dokku-postgres.git"
}

resource "dokku_postgres" "demo" {
  service_name = "demo-service"

  depends_on = [
    dokku_plugin.postgres
  ]
}

resource "dokku_app" "demo" {
  app_name = "demo-app"
}

# Requires plugin to be installed
# https://github.com/dokku/dokku-postgres
resource "dokku_postgres_link" "demo" {
  app_name     = "demo-app"
  service_name = "demo-service"
  # you'll have "FOOBAR_URL" env var in your app, pointing to postgres database
  alias = "FOOBAR"

  depends_on = [
    dokku_app.demo,
    dokku_postgres.demo
  ]
}
