# Terraform Provider for Dokku (Terraform Plugin Framework)

This repository is a [Terraform](https://www.terraform.io) provider for [Dokku](https://dokku.com/).

For now only subset of dokku features are supported now.

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

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.
