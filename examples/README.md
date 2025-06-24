# Examples of Ephemeral Access Extension Plugin for Service Now

This directory contains some examples that might help you when you either want
to use the plugin or when you want to test or enhance the plugin.

## Use of the plugin

In the beginning it might be hard to understand which manifests should be
created to get your implementation working. For this goal, I created two
examples:

### Service Now scripts

You can add a CI and a change manually in the Service Now console, you can also
use these Service Now scripts. This will work faster and is useful when your
development environment of ServiceNow has been erased and you have to start from
zero to add your CI and change(s). It is also handy to add a new change fast,
without having to walk through all the ServiceNow screens.

### AWS example

This example will deploy a three node Kubernetes cluster, with ArgoCD and the
plugin installed, on AWS.

### Kubernetes ephemeral access extension

In this directory you will find a complete explaination of the Ephemeral Access
Extension (without the plugin).

## Test or enhance the plugin

I'm using CloudFormation templates to test or enhance the plugin. You can find
more information in [this README.md](./dev-test-build-release/README.md) in the
dev-test-build-release directory.
