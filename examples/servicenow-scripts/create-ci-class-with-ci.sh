#!/bin/bash

URL=$1
USERNAME=$2
PASSWORD=$3

CLASS_LABEL="Kubernetes Application"
CLASS_NAME="u_cmdb_ci_kubernetes_application"
DESCRIPTION="Inserted by ServiceNow script create-ci-class.sh"
CI_NAME="app-demoapp"

# Install status 
#
#   Installed = "1"         (*)
#   On order = "2"
#	In maintenance = "3"    (*)
#	Pending Install = "4"   (*)
#	Pending Repair = "5"    (*)
#   In Stock = "6"
#   Retired = "7"
#   Stolen = "8"
#
# (*) = valid for the plugin:

INSTALL_STATUS="1"

function check_parameters() {
    if test "${URL}" == ""
    then
        echo "Usage  : create-ci-class-with-ci.sh url service-now-username service-now-password"
        echo "Example: create-ci-class-with-ci.sh https://dev123456.service-now.com admin a%B12CdE"
        echo ""
        echo "Error: no host, username and password provided"
        exit 1
    fi

    if test "${USERNAME}" == ""
    then
        echo "Usage  : create-ci-class-with-ci.sh url service-now-username service-now-password"
        echo "Example: create-ci-class-with-ci.sh https://dev123456.service-now.com admin a%B12CdE"
        echo ""
        echo "Error: no username and password provided"
        exit 1
    fi

    if test "${PASSWORD}" == ""
    then
        echo "Usage  : create-ci-class-with-ci.sh service-now-username service-now-password"
        echo "Example: create-ci-class-with-ci.sh https://dev123456.service-now.com admin a%B12CdE"
        echo ""
        echo "Error: no password provided"
        exit 1
    fi
}

function check_class() {
    curl "${URL}/api/now/table/cmdb_class_info?class=${CLASS_NAME}&sysparm_fields=class" \
        --request GET \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
   	    --user "${USERNAME}":"${PASSWORD}" | jq '.result[].class' | awk -F'"' '{print $2}'
}

function add_class() {

    OUTPUT=$(curl "${URL}/api/now/cimodel/class" \
        --request POST \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
        --user "${USERNAME}":"${PASSWORD}" \
        --data "{\"name\":\"${CLASS_NAME}\",\"label\":\"${CLASS_LABEL}\",\"description\":\"${DESCRIPTION}\",\"super_class\":\"cmdb_ci\",\"icon\":\"c2442dd69fb00200eb3919eb552e7011\",\"is_extendable\":false,\"principal_class\":false,\"managed_by_group\":\"\",\"columns\":[],\"identifier\":{\"dependencies\":[]},\"reconciliation\":{\"reconciliation_definitions\":[],\"datasource_refreshness\":[]}}" )

    if test "${OUTPUT}" != "{\"result\":[]}"
    then
        echo "Adding class was not succesful, change script or parameters and try again. Error: ${OUTPUT}"
        exit 1
    else
        echo "Class succesfully added to ServiceNow"
    fi
}

function check_ci() {
    curl "${URL}/api/now/v1/table/${CLASS_NAME}?name=${CI_NAME}&sysparm_fields=name" \
        --request GET \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
        --user "${USERNAME}":"${PASSWORD}" | jq '.result[].name' 2>/dev/null | awk -F'"' '{print $2}'
}

function add_ci() {
    OUTPUT=$(curl "${URL}/api/now/v1/table/${CLASS_NAME}" \
        --request POST \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
        --user "${USERNAME}":"${PASSWORD}" \
        --data "{\"name\":\"${CI_NAME}\", \"install_status\":\"${INSTALL_STATUS}\"}")
    SYS_ID=$(echo "${OUTPUT}" | jq '.result.sys_id' | awk -F'"' '{print $2}')
    
    if test "$SYS_ID" == ""
    then
       echo "Error in adding CI: $OUTPUT"
       exit 1
    fi
}

check_parameters
RESULT=$(check_class)
if test "${RESULT}" != "${CLASS_NAME}"
then
    echo "Class ${CLASS_NAME} doesn't exist yet, adding it..."
    add_class
else
    echo "Class ${CLASS_NAME} already exists, adding ${CI_NAME} under existing class"
fi

RESULT=$(check_ci)
if test "${RESULT}" != "${CI_NAME}"
then
  echo "CI ${CI_NAME} doesn't exist yet, adding it..."
  add_ci
else
  echo "CI ${CI_NAME} already exists"
  exit 1
fi
