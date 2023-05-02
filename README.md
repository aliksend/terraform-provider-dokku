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

Add the provider and host settings to your terraform block.
The SSH key should be that of a [dokku user](https://dokku.com/docs/deployment/user-management/). Dokku users have dokku set as a forced command - the provider will not attempt to explicitly specify the dokku binary over SSH.

```hcl
terraform {
  required_providers {
    dokku = {
      source  = "registry.terraform.io/aliksend/dokku"
    }
  }
}

provider "dokku" {
  ssh_host = "dokku.me"

  # optional
  ssh_user = "dokku"
  ssh_port = 22
  ssh_cert = "~/.ssh/id_rsa"

  # to support copying from local to host
  scp_user = "root"
  scp_cert = "~/.ssh/root_rsa"
}
```

[Documentation](docs/index.md)

### Deploy using push from git repository (simplest way)

Example .gitlab-ci.yml
```yml
stages:
  - terraform
  - deploy

variables:
  SSH_HOST: __YOUR_HOST__
  APP_NAME: __YOUR_APP__
  TF_STATE_ADDRESS: "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/terraform/state/main"

terraform:
  image:
    name: hashicorp/terraform:light
    entrypoint: ['']
  stage: terraform
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
      -var ssh_cert="$SSH_PRIVATE_KEY"

dokku_deploy:
  image: ilyasemenov/gitlab-ci-git-push
  stage: deploy
  only:
    - master
  script:
    - git-push ssh://dokku@$SSH_HOST/$APP_NAME
```

You need to have gitlab variable SSH_PRIVATE_KEY with private key, added in step 2.

Example terraform configuration
```terraform
variable "ssh_cert" {
  type = string
  description = "SSH cert"
  default = "~/.ssh/id_rsa"
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

  deploy = null
}
```

### Deploy using sync with git repository

Example .gitlab-ci.yml
```yml
stages:
  - deploy

variables:
  TF_STATE_ADDRESS: "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/terraform/state/main"

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
      -var ssh_cert="$SSH_PRIVATE_KEY"
      -var git_repository="$CI_REPOSITORY_URL"
      -var git_repository_ref="$CI_COMMIT_SHA"
```

You need to have gitlab variable SSH_PRIVATE_KEY with private key, added in step 2.

As long as built-in gitlab env var CI_REPOSITORY_URL contains credentials you don't need to provide it explicitly.

Example terraform configuration
```terraform
variable "ssh_cert" {
  type = string
  description = "SSH cert"
  default = "~/.ssh/id_rsa"
}

variable "git_repository" {
  type = string
  description = "Git repository to sync with"
}

variable "git_repository_ref" {
  type = string
  description = "Ref in git repository to sync with"
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
    type = "git_repository"
    git_repository = var.git_repository
    git_repository_ref = var.git_repository_ref
  }
}
```

### Deploy using docker image

Example .gitlabci.yml
```yml
stages:
  - build
  - deploy

variables:
  TF_STATE_ADDRESS: "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/terraform/state/main"

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
variable "ssh_cert" {
  type = string
  description = "SSH cert"
  default = "~/.ssh/id_rsa"
}

variable "docker_image" {
  type = string
  description = "Docker image to deploy"
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
go install .
```
