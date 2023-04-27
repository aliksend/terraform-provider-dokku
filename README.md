# Terraform Provider for Dokku (Terraform Plugin Framework)

This repository is a [Terraform](https://www.terraform.io) provider for [Dokku](https://dokku.com/).

For now only subset of dokku features are supported now.

## Using the provider

1. [Set up dokku](https://dokku.com/docs/getting-started/installation/#installing-the-latest-stable-version) or [Upgrade dokku](https://dokku.com/docs/getting-started/upgrading/) on installations with prebuilt dokku (like on DO)

2. Set up SSH keys
```bash
# ON LOCAL PC
# Set up publickey auth
ssh-copy-id user@IP

# ON VPS
# Add key to dokku
# You can change "admin" to any preferred username, describing who you are related to this server instance
cat ~/.ssh/authorized_keys | dokku ssh-keys:add admin
```

[Documentation](docs/index.md)

Example .gitlabci.yml
```yml
stages:
  - build
  - deploy

variables:
  TF_STATE_ADDRESS: "https://gitlab.com/api/v4/projects/${CI_PROJECT_ID}/terraform/state/main"

build:
  image: docker:stable
  stage: build
  services:
    - docker:dind
  only:
    - master
  variables:
    DOCKER_HOST: tcp://docker:2375
    DOCKER_DRIVER: overlay2
  script:
    - docker login -u gitlab-ci-token -p $CI_JOB_TOKEN registry.gitlab.com
    - docker pull $CI_REGISTRY_IMAGE:latest || true
    - docker build --cache-from $CI_REGISTRY_IMAGE:latest --tag $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA --tag $CI_REGISTRY_IMAGE:latest .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - docker push $CI_REGISTRY_IMAGE:latest


dokku_deploy:
  image:
    name: hashicorp/terraform:light
    entrypoint: ['']
  stage: deploy
  only:
    - master
  script:
    - terraform version
    - terraform init
      -reconfigure
      -backend-config="address=${TF_STATE_ADDRESS}"
      -backend-config="lock_address=${TF_STATE_ADDRESS}/lock"
      -backend-config="unlock_address=${TF_STATE_ADDRESS}/lock"
      -backend-config="username=gitlab-ci-token"
      -backend-config="password=$CI_JOB_TOKEN"
      -backend-config="lock_method=POST"
      -backend-config="unlock_method=DELETE"
      -backend-config="retry_wait_min=5"
    - terraform apply
      -input=false
      -auto-approve
      -var docker_image="$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA"
      -var ssh_cert="$SSH_PRIVATE_KEY"
      -var docker_image_registry_login="gitlab-ci-token"
      -var docker_image_registry_password="$CI_JOB_TOKEN"

```

You need to have gitlab variable SSH_PRIVATE_KEY with private key, added in step 2.

Example terraform configuration
```terraform
variable "docker_image" {
  type = string
  description = "Docker image to deploy"
}

variable "ssh_cert" {
  type = string
  description = "SSH cert"
  default = "~/.ssh/id_rsa"
}

variable "docker_image_registry_login" {
  type = string
  description = "Login for Registry of your docker image"
}

variable "docker_image_registry_password" {
  type = string
  description = "Password for Registry of your docker image"
}

terraform {
  required_providers {
    dokku = {
      source = "registry.terraform.io/aliksend/dokku"
    }
  }

  backend "http" {}
}

provider "dokku" {
  ...
  ssh_cert = var.ssh_cert
}

resource "dokku_app" "yourname" {
  ...

  deploy = {
    type = "docker_image"
    login = var.docker_image_registry_login
    password = var.docker_image_registry_password
    docker_image = var.docker_image
  }
}
```

# Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate ./...`.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```
