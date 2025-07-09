# Variables for complex app configuration test
variable "dokku_host" {
  description = "Dokku server hostname or IP"
  type        = string
  default     = "localhost"
}

variable "dokku_port" {
  description = "SSH port for Dokku server"
  type        = number
  default     = 3022
}

variable "ssh_private_key" {
  description = "SSH private key content"
  type        = string
  sensitive   = true
}

variable "app_name" {
  description = "Name of the Dokku app to create"
  type        = string
  default     = "complex-test-app"
}

variable "app_config" {
  description = "Application configuration variables"
  type        = map(string)
  default = {
    ENV      = "prod"
    APP_NAME = "" # Will be set to var.app_name in terraform.tfvars
    NODE_ENV = "production"
    PORT     = "3000"
    DEBUG    = "false"
  }
}

variable "docker_image" {
  description = "Docker image to run"
  default = "jmalloc/echoserver"
  type = string
}
