resource "dokku_plugin" "letsencrypt" {
  name = "letsencrypt"
  url  = "https://github.com/dokku/dokku-letsencrypt.git"
}


# Requires plugin to be installed
# https://github.com/dokku/dokku-letsencrypt
resource "dokku_letsencrypt" "demo" {
  app_name = "demo"
  email    = "demo@example.com"

  depends_on = [
    dokku_app.demo,
    dokku_plugin.letsencrypt
  ]
}
