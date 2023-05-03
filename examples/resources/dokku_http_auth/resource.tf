resource "dokku_http_auth" "demo" {
  app_name = "demo"

  users = {
    foo = {
      password = "bar"
    }
  }
}
