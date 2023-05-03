resource "dokku_letsencrypt" "demo" {
  app_name = "demo"
  email    = "demo@example.com"
}
