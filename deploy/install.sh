#!/bin/bash
SERVICE_NOW_URL=https://dev123456.service-now.com
TIMEZONE="Europe/Amsterdam"

PLUGIN_VERSION=REPLACE_BY_NEWEST_VERSION
PLUGIN_IMAGE=frederiquer/argocd-ephemeral-access-plugin-servicenow:${PLUGIN_VERSION}
MANIFEST_FILENAME="controller-patch.yaml"

kubectl apply -f "https://github.com/FrederiqueRetsema/argocd-ephemeral-access-plugin-servicenow/releases/download/${PLUGIN_VERSION}/install.yaml"
kubectl apply -f "https://github.com/FrederiqueRetsema/argocd-ephemeral-access-plugin-servicenow/releases/download/${PLUGIN_VERSION}/controller-role.yaml"

cd /tmp || exit

curl -OL "https://github.com/FrederiqueRetsema/argocd-ephemeral-access-plugin-servicenow/releases/download/${PLUGIN_VERSION}/${MANIFEST_FILENAME}"

sed -i "s^CHANGE_THIS_TO_POINT_TO_YOUR_SERVICENOW_URL^${SERVICE_NOW_URL}^" ${MANIFEST_FILENAME}
sed -i "s^CHANGE_THIS_TO_THE_TIMEZONE_IN_SERVICE_NOW^${TIMEZONE}^" "${MANIFEST_FILENAME}"
sed -i "s^CHANGE_THIS_TO_POINT_TO_YOUR_PLUGIN_IMAGE^${PLUGIN_IMAGE}^" "${MANIFEST_FILENAME}"
kubectl patch deployment controller -n argocd-ephemeral-access --patch-file "${MANIFEST_FILENAME}"

cd - || exit
