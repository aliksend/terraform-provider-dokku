terraform {
  required_providers {
    dokku = {
      source = "registry.terraform.io/aliksend/dokku"
    }
  }
}

# simple configuration
provider "dokku" {
  ssh_host = "127.0.0.1"
}

# custom certificate
# raw certificate can be provided
variable "ssh_cert" {
  type        = string
  description = "SSH cert"
  default     = "~/.ssh/id_rsa"
}

provider "dokku" {
  ssh_host     = "127.0.0.1"
  ssh_port     = 2222
  ssh_cert     = var.ssh_cert
  ssh_host_key = "127.0.0.1 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCql...Dq+Nnpue8="
}
