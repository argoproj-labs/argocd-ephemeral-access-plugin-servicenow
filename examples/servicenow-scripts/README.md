# ServiceNow scripts

In this directory you will find scripts to add a new CI and a new change fast.

The scripts use the ServiceNow URL, username and password as parameters for the
script. You can use the admin username and password for this.

Example:

`create-ci-class-with-ci.sh https://dev123456.service-now.com admin a%B12CdE`

It is wise to change the contents of the scripts to what you want, the current
content works for me.

## create-ci-class-with-ci.sh

Will add a CI in a CI Class. When the CI Class already exists, it will add the
CI to the existing class. Please mind, that ServiceNow allows multiple different
CIs to have the same name and be part of the same CI Class. The plugin will
assume that the CI names are unique. The plugin will pick the first CI with a
certain name.

## create-change.sh

Will create a change by following all steps that you would do in the ServiceNow
portal. You can give the change number that you want and the change will be
created from one hour before the current date/time until (more than) one hour
after the current date/time.
