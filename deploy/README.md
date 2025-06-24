# Install directory

This directory contains the scripts to create a release.

## tag.sh

Used to create a release in GitHub. Will replace the version numbers in the
install.sh script by the correct version number. Will also add the HEADER.md
file on top of the changes. Also adds the files that are used by the install
script to the release.

## install.sh

Will change the configuration files based on the local settings for f.e.
the ServiceNow URL.
