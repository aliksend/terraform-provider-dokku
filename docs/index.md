---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "dokku Provider"
subcategory: ""
description: |-
  Interact with dokku
---

# dokku Provider

Interact with dokku

## Example Usage

```terraform
provider "dokku" {
  ssh_host = "127.0.0.1"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `ssh_host` (String) Host to connect to

### Optional

- `fail_on_untested_version` (Boolean)
- `log_ssh_commands` (Boolean) Print SSH commands with ERROR verbose
- `scp_cert` (String) Cert for username that will be used to copy files to server. Default: ssh_cert value
- `scp_user` (String) Username that will be used to copy files to server. If not set then scp feature will be disabled
- `ssh_cert` (String) Certificate to use. Supported formats:
- file:/a or /a or ./a or ~/a - use provided value as path to certificate file
- env:ABCD or $ABCD - use env var ABCD
- raw:----.. or ----... - use provided value as raw certificate
Default: ~/.ssh/id_rsa
- `ssh_port` (Number) Port to connect to. Default: 22
- `ssh_user` (String) Username to use. Default: dokku
