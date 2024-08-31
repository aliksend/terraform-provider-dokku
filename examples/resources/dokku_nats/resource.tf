resource "dokku_plugin" "nats" {
  name = "nats"
  url  = "https://github.com/dokku/dokku-nats.git"
}

# Requires plugin to be installed
# https://github.com/dokku/dokku-nats
resource "dokku_nats" "demo" {
  service_name = "demo"
  # config options for nats (optional)
  config_options = "--jetstream"
  # hots:port or port to expose service to host machine (optional)
  expose = "1234"

  depends_on = [
    dokku_plugin.nats
  ]
}
