# Development and test environment for the plugin

I'm using AWS for testing and developing the plugin. Two AWS CloudFormation
templates proved to be helpful:

## aws-dev

This template will install a three node Kubernetes cluster, with ArgoCD and the
plugin installed. The plugin is created from scratch, into a temporary AWS ECR
container repository. There are basic one-letter commands to rebuild the plugin,
delete the current controller pod, read the logs from the controller pod etc.

## aws-pre-install-plugin

This template will install a three node Kubernetes cluster, with ArgoCD
installed. The plugin is not installed. This is useful to test a first-time
install of the plugin, based on the public Docker Hub container image.
