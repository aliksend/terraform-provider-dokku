terraform {
  required_providers {
    dokku = {
      source = "localhost/providers/dokku"
    }
  }
}

provider "dokku" {
  ssh_host                = var.dokku_host
  ssh_port                = var.dokku_port
  ssh_user                = "dokku"
  ssh_cert                = var.ssh_private_key
  ssh_skip_host_key_check = true
  log_ssh_commands        = true
}

# Simple dokku_app resource with minimal configuration
resource "dokku_app" "simple_app" {
  app_name = var.app_name

  config = {
    PORT = "5000"
  }

  # Basic deployment with docker image
  deploy = {
    type         = "docker_image"
    docker_image = "jmalloc/echo-server"
  }

  # Configure ports for proper HTTP routing
  # jmalloc/echo-server serves on port 8080, so we map external port 80 to container port 8080
  ports = {
    80 = {
      scheme         = "http"
      container_port = 5000 # This is the port jmalloc/echo-server actually runs on
    }
  }

  domains = [
    "${var.app_name}.dokku.test",
    "dokku.test"
  ]
}

# Test outputs
output "app_name" {
  value = dokku_app.simple_app.app_name
}
