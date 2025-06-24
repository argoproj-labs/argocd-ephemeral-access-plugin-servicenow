# Settings

## Environment variables

Most settings are configured using environment variables in the
controller-deployment. The environment variables are by default all configured
using their default values:

| Environment variable                 | Default value           |
|--------------------------------------|-------------------------|
| EPHEMERAL_ACCESS_EXTENSION_NAMESPACE | argocd-ephemeral-access |
| SERVICENOW_SECRET_NAME               | servicenow-secret       |
| SERVICENOW_URL                       | no default              |
| TIME_WINDOW_CHANGES_DAYS             | 7                       |
| TIMEZONE                             | UTC                     |
| CI_LABEL                             | ciName                  |

### EPHEMERAL_ACCESS_EXTENSION_NAMESPACE

This is the namespace where the Ephemeral Access Extension is installed.

### SERVICENOW_SECRET_NAME

The secret name where the username and password of the ServiceNow user are
stored. The secret should be installed in the
`EPHEMERAL_ACCESS_EXTENSION_NAMESPACE` namespace

### SERVICENOW_URL

The URL of ServiceNow, in the format `https://your-instance.service-now.com`.

### TIME_WINDOW_CHANGES_DAYS

Time window to find relevant changes. This is needed because currently (*) there
is no way to pass the change number in the environment. This is solved by
searching for all changes in the time window of x days before and after the
current day. Example: in the default settings, the plugin will look at all
changes from one week before the current day until one week after the current
day. The time is not taken into account, it will search from 00:00:00 on the
start date until 23:59:59 of the end date. When the start date is before this
moment _or_ the end date is after this moment, the change is not found.

(*) See also the discussion via
[issue 16](https://github.com/FrederiqueRetsema/argocd-ephemeral-access-plugin-servicenow/issues/16)

### TIMEZONE

Time zone of the user in ServiceNow. The time zone of the plugin should match
the time zone of the user in ServiceNow, otherwise incorrect conclusions about
start time and end time are drawn.

### CI_LABEL

Name of the label in the application that indicates what the application name in
ServiceNow is.

## Config maps

There is one config map that is relevant to this plugin: it is the
`controller-cm` config map (that is created already by the Ephemeral Access
Extension). In this configmap you can configure both the log level (for both the
controller itself and the plugin) and the exclusion roles.

Example configmap:

```Manifest
apiVersion: v1
kind: ConfigMap
metadata:
  name: controller-cm
  namespace: argocd-ephemeral-access
data:
  controller.log.level: debug
  exclusion-roles: |
    incidentmanager
```

### Log level

The log level can be configured via the keyword `controller.log.level`. You can
use debug, info, warn and error.

### Exclusion roles

You can configure the exclusion roles in this list. Please mind, that
exclusion-roles are roles that are defined in the ArgoCD Ephemeral Access
Extension, they are not refering to OIDC groups. In this way, one person can use
both a normal role (where a CI and a change are used) and an exclusion role
(where one gets access directly).
