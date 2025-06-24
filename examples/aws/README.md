# AWS Development template

## Introduction

The CloudFormation template in this directory can be used to look at the
Ephemeral Access Extension plugin for ServiceNow from an end user perspective.
The cloudformation template will pull a specific version of the plugin from
Docker Hub when the Virtual Machine is started. The virtual machine contains
tools to rebuild the plugin and deploy new manifest files as well.

## Installation

### Start and stop scripts

First log in to your account, then use the `start-k8s.sh` and `stop-k8s.sh`
scripts to start and stop the development environment. You need to change some
parameters in these scripts, for example the `profile name` of your environment.

The `consultant name` is used for both the stack name in AWS and for the user
name in the virtual machine.

The `consultant email` is used for AWS Congito: it will send you an email with
the initial password.

`Group name` is used for the AWS Cognito group name that will be used for this
example. It will also be used in the Virtual Machine for the Kubernetes
manifests to give this group permissions within the Ephemeral Access Extension.

`Hosted zone name` is the hosted zone name in Route53 within your AWS
environment. It is used to access the ArgoCD website for the cluster.

`Certificate ID` is the ID of the certificate that will be used for the argocd
website. It can be a star certificate (f.e. *.sandbox2.prutsforce.nl) or a
specific certificate (f.e. argocd.sandbox2.prutsforce.nl).

`Default password` is the initial password for your user on the nodes. Please
change this for your own environment.

`ServiceNow secret name` is the name that you want to use for the secret that
contains the username and password for the ServiceNow environment. You can
create one or find it back in the AWS Certificate Manager service:

![certificate](./acm-certificate.png)

`ServiceNow URL` is the URL to your ServiceNow environment. You can use a free
developer subscription if you don't have a ServiceNow environment yourself,
request one at <https://developer.servicenow.com/dev.do>.

`Local Timezone` is the timezone which is used in ServiceNow. The plugin should
use the same timezone as ServiceNow (the timezone of the server can be different).

### Initialization in the virtual machine

After you log on, you need to install the secret for ServiceNow. This can be
done with the following command (change the password):

```kubernetes
kubectl create secret -n argocd-ephemeral-access generic servicenow-secret \
  --from-literal=username=admin \
  --from-literal=password=ABc@defC1D3%
```
