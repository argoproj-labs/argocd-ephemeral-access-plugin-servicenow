#!/bin/bash

TAG=$1

if test "${TAG}" == ""
then
   echo "Usage  : scriptname.sh tag"
   echo "Example: scriptname.sh v0.1.2"
   exit 1
fi 

cp HEADER.md /tmp
sed -i "s/REPLACE_BY_NEWEST_VERSION/${TAG}/" /tmp/HEADER.md

cp install.sh /tmp
sed -i "s/REPLACE_BY_NEWEST_VERSION/${TAG}/" /tmp/install.sh

gh release create "${TAG}" --fail-on-no-commits --generate-notes --notes-file /tmp/HEADER.md --latest --title "${TAG}"
gh release upload "${TAG}" /tmp/install.sh --clobber
gh release upload "${TAG}" ../LICENSE --clobber
gh release upload "${TAG}" ../manifests/plugin/controller-patch.yaml --clobber
gh release upload "${TAG}" ../manifests/plugin/controller-role.yaml --clobber
gh release upload "${TAG}" ../manifests/install.yaml --clobber
