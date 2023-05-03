resource "dokku_app" "demo" {
  app_name = "demo"
}

resource "dokku_app" "demo2" {
  app_name = "demo2"

  config = {
    foo = "bar"
  }

  storage = {
    uploads = {
      mount_path = "/app/uploads"
    }
    "/var/log" = {
      mount_path = "/var/log"
    }
  }

  proxy_ports = {
    80 = {
      scheme         = "http"
      container_port = 5000
    }
  }

  domains = ["example.com"]

  docker_options = {
    "--label demo" = {
      phase = ["deploy"]
    }
  }
}
