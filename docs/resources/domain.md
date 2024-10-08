---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "dokku_domain Resource - terraform-provider-dokku"
subcategory: ""
description: |-
  For global domains setup
  For app-specific domains use app_resource.domains attribute
  https://dokku.com/docs/configuration/domains/
---

# dokku_domain (Resource)

For global domains setup
  For app-specific domains use app_resource.domains attribute
  https://dokku.com/docs/configuration/domains/

## Example Usage

```terraform
# For global domains setup
# For app-specific domains use app_resource.domains attribute
# https://dokku.com/docs/configuration/domains/
resource "dokku_domain" "demo" {
  domain = "example.com"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `domain` (String) Domain to use

## Import

Import is supported using the following syntax:

```shell
# dokku_domain can be imported by specifying the app name
terraform import dokku_domain.demo 'example.com'
```
