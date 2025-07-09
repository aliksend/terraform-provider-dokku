# Variables for simple test configuration
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
  default     = "simple-test-app"
}

variable "docker_image" {
  description = "Docker image to run"
  default = "jmalloc/echo-server"
  type = string
}
