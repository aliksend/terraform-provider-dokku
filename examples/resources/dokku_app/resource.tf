resource "dokku_app" "demo" {
  app_name = "demo"
}

resource "dokku_app" "demo2" {
  app_name = "demo2"

  # https://dokku.com/docs/configuration/environment-variables/
  config = {
    foo = "bar"
  }

  # https://dokku.com/docs/deployment/zero-downtime-deploys/
  checks = {
    status = "disabled"
  }

  # https://dokku.com/docs/advanced-usage/persistent-storage/
  storage = {
    uploads = {
      mount_path = "/app/uploads"
    }
    "/var/log" = {
      mount_path = "/var/log"
    }
    config = {
      mount_path = "/app/config"
      # copy local directory "./config" to host directory, that will be mounted as "/app/config"
      local_directory = "./config"
    }
  }

  # DEPRECATED use ports instead
  proxy_ports = {
    80 = {
      scheme         = "http"
      container_port = 5000
    }
  }

  # https://dokku.com/docs/networking/port-management/
  ports = {
    80 = {
      scheme         = "http"
      container_port = 5000
    }
  }

  # https://dokku.com/docs/configuration/domains/
  domains = ["example.com"]

  # https://dokku.com/docs/advanced-usage/docker-options/
  docker_options = {
    "--label demo" = {
      phase = ["deploy"]
    }
  }

  # https://dokku.com/docs/networking/network/
  networks = {
    attach_post_create = "internal"
  }

  # https://dokku.com/docs/deployment/methods/git/
  # https://dokku.com/docs/deployment/methods/image/
  # https://dokku.com/docs/deployment/methods/archive/
  deploy = {
    type         = "docker_image"
    login        = var.docker_image_registry_login
    password     = var.docker_image_registry_password
    docker_image = var.docker_image
  }
}
