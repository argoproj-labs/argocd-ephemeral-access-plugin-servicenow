#!/bin/bash
PROFILE="xforce-sandbox2"
CONSULTANT_NAME="frederique"                                # must be lowercase
CONSULTANT_EMAIL="frederique.retsema@conclusionxforce.nl"
GROUP_NAME="xforce-admins"
HOSTED_ZONE_NAME="sandbox2.prutsforce.nl"                  # Without a dot, dot will be added where necessary
CERTIFICATE_ID="867a7fae-455f-4e3c-8efd-7bf2d37fe990"      # *.sandbox2.prutsforce.nl
DEFAULT_PASSWORD="ConclusionXforce2025!"
SERVICENOW_SECRET_NAME="servicenow-secret"
SERVICENOW_URL="https://dev123456.service-now.com"
LOCAL_TIMEZONE="Europe/Amsterdam"
ARGOCD_NAMESPACE="argocd"

# The cloudformation template is rather big, CloudFormation wants the template in a bucker.
# The default name is the consultantname + "-" + the profile, in my case frederique-xforce-sandbox1
# The bucket must exist before the script is started.

BUCKET_NAME="${CONSULTANT_NAME}-${PROFILE}"

STACK_NAME="${CONSULTANT_NAME}-k8s"

echo "STACK_NAME=${STACK_NAME}"
echo "Bucketname=${BUCKET_NAME}"
echo ""
echo "Create the bucket first with"
echo "aws s3 mb s3://$BUCKET_NAME --profile $PROFILE"
echo "when you get errors in the command below"
echo ""
echo "Deploy started at $(date +%H:%M:%S), it will take about 15 minutes to finish"

aws cloudformation deploy \
       --stack-name "${STACK_NAME}" \
       --template-file "./cloudformation.yaml" \
       --parameter-overrides \
           ConsultantName="${CONSULTANT_NAME}" \
           ConsultantEmail="${CONSULTANT_EMAIL}" \
           GroupName="${GROUP_NAME}" \
           DefaultPassword="${DEFAULT_PASSWORD}" \
           HostedZoneName="${HOSTED_ZONE_NAME}" \
           CertificateId="${CERTIFICATE_ID}" \
           ServiceNowUrl="${SERVICENOW_URL}" \
           ServiceNowSecretName="${SERVICENOW_SECRET_NAME}" \
           LocalTimezone="${LOCAL_TIMEZONE}" \
           ArgoCDNamespace="${ARGOCD_NAMESPACE}" \
        --capabilities "CAPABILITY_IAM" \
        --s3-bucket "${BUCKET_NAME}" \
        --profile "${PROFILE}"

echo "Deploy finished at $(date +%H:%M:%S)"

IDS=$(aws cloudformation describe-stacks --stack-name ${STACK_NAME} --profile "${PROFILE}")
ID_CONTROL=$(echo "$IDS" | jq '.Stacks[0].Outputs[] | select(.OutputKey=="ControlNodeId") | .OutputValue' | awk -F'"' '{print $2}')

echo "---"
echo "Website argo CD: https://argocd.${HOSTED_ZONE_NAME}"
echo "Command to log on to the control node:"
echo "   aws ssm start-session --target ${ID_CONTROL} --profile ${PROFILE}"
echo ""
echo "On the control node use the following commands to go to the right account:"
echo "   sudo -i"
echo "   su - ${CONSULTANT_NAME}"
