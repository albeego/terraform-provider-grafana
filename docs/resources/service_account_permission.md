---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "grafana_service_account_permission Resource - terraform-provider-grafana"
subcategory: "Grafana OSS"
description: |-
  Note: This resource is available from Grafana 9.2.4 onwards.
  Official documentation https://grafana.com/docs/grafana/latest/administration/service-accounts/#manage-users-and-teams-permissions-for-a-service-account-in-grafana
---

# grafana_service_account_permission (Resource)

**Note:** This resource is available from Grafana 9.2.4 onwards.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/#manage-users-and-teams-permissions-for-a-service-account-in-grafana)

## Example Usage

```terraform
resource "grafana_service_account" "test" {
  name        = "sa-terraform-test"
  role        = "Editor"
  is_disabled = false
}

resource "grafana_team" "test_team" {
  name = "tf_test_team"
}

resource "grafana_user" "test_user" {
  email    = "tf_user@test.com"
  login    = "tf_user@test.com"
  password = "password"
}

resource "grafana_service_account_permission" "test_permissions" {
  service_account_id = grafana_service_account.test.id

  permissions {
    user_id    = grafana_user.test_user.id
    permission = "Edit"
  }
  permissions {
    team_id    = grafana_team.test_team.id
    permission = "Admin"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `permissions` (Block Set, Min: 1) The permission items to add/update. Items that are omitted from the list will be removed. (see [below for nested schema](#nestedblock--permissions))
- `service_account_id` (String) The id of the service account.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--permissions"></a>
### Nested Schema for `permissions`

Required:

- `permission` (String) Permission to associate with item. Must be `Edit` or `Admin`.

Optional:

- `team_id` (Number) ID of the team to manage permissions for. Specify either this or `user_id`. Defaults to `0`.
- `user_id` (Number) ID of the user to manage permissions for. Specify either this or `team_id`. Defaults to `0`.
