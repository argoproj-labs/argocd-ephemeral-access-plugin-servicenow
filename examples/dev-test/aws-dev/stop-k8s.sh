#!/bin/bash
PROFILE="xforce-sandbox2"
CONSULTANT_NAME="frederique"

NAMES=$(aws cloudformation describe-stacks --profile "${PROFILE}" | jq '.Stacks[].StackName' | awk -F'"' '{print $2}' | grep "${CONSULTANT_NAME}-k8s")

for NAME in ${NAMES}
do
  echo "Delete stack ${NAME}"
  aws cloudformation delete-stack --stack-name "${NAME}" --profile "${PROFILE}"
done
