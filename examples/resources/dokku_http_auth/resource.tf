variable "docker_image" {
  description = "Docker image to deploy"
  type        = string
}

resource "dokku_app" "app" {
  app_name = "demo"
  deploy = {
    type         = "docker_image"
    docker_image = var.docker_image
  }

  config = {
    PORT = "5000"
  }

  ports = {
    80 = {
      scheme         = "http"
      container_port = 5000
    }
  }
}

resource "dokku_http_auth" "demo" {
  app_name = dokku_app.app.app_name

  users = {
    test_user = {
      password = "test_password"
    }
  }
}
