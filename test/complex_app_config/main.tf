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

# Complex dokku_app resource with advanced configuration
resource "dokku_app" "complex_app" {
  app_name = var.app_name

  config = var.app_config

  # Use a set of domains
  domains = [
    "${var.app_name}.dokku.test",
    "api.${var.app_name}.dokku.test"
  ]

  # Configure deployment
  deploy = {
    type         = "docker_image"
    docker_image = var.docker_image
  }

  # Configure ports - keys are host ports as strings, values have scheme and container_port as strings
  ports = {
    "5000" = {
      scheme         = "http"
      container_port = "5000"
    }
  }
}

# Test outputs
output "app_name" {
  value = dokku_app.complex_app.app_name
}

output "app_domains" {
  value = dokku_app.complex_app.domains
}

output "app_config" {
  value     = dokku_app.complex_app.config
  sensitive = true
}
