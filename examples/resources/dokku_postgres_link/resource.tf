resource "dokku_postgres" "demo" {
  service_name = "demo-service"
}

resource "dokku_app" "demo" {
  app_name = "demo-app"
}

resource "dokku_postgres_link" "demo" {
  app_name     = "demo-app"
  service_name = "demo-service"
  alias        = "DATABASE"
}
