#!/bin/bash

URL=$1
USERNAME=$2
PASSWORD=$3

CHANGE_NUMBER="CHG0080011"
CMDB_CI="app-demoapp"
SHORT_DESCRIPTION="Test Ephemeral Access Plugin for ServiceNow"

ONE_HOUR_AGO_UTC=$(date -u -d "1 hour ago" +"%Y-%m-%d %H:%M:%S")
MORE_THAN_ONE_HOUR_FROM_NOW_UTC=$(date -u -d "2 hour" +"%Y-%m-%d %H:00:00")

function check_parameters() {
    if test "${URL}" == ""
    then
        echo "Usage  : create-change.sh url service-now-username service-now-password"
        echo "Example: create-change.sh https://dev123456.service-now.com admin a%B12CdE"
        echo ""
        echo "Error: no host, username and password provided"
        exit 1
    fi

    if test "${USERNAME}" == ""
    then
        echo "Usage  : create-change.sh url service-now-username service-now-password"
        echo "Example: create-change.sh https://dev123456.service-now.com admin a%B12CdE"
        echo ""
        echo "Error: no username and password provided"
        exit 1
    fi

    if test "${PASSWORD}" == ""
    then
        echo "Usage  : create-change.sh service-now-username service-now-password"
        echo "Example: create-change.sh https://dev123456.service-now.com admin a%B12CdE"
        echo ""
        echo "Error: no password provided"
        exit 1
    fi
}

function find_sys_id_assignment_group_application_development() {
    curl "${URL}/api/now/table/sys_user_group?name=Application%20Development" \
        --request GET \
        --header "Accept:application/json" \
        --user "${USERNAME}":"${PASSWORD}" | jq .result[].sys_id | awk -F'"' '{print $2}'
}

function check_change() {
    curl "${URL}/api/now/table/change_request?number=${CHANGE_NUMBER}&sysparm_fields=number" \
        --request GET \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
        --user "${USERNAME}":"${PASSWORD}" | jq '.result[].number' 2>/dev/null | head -n 1 | awk -F'"' '{print $2}'
}

function add_change() {
    SYS_ID_ASSIGNMENT_GROUP=$1

    OUTPUT=$(curl "${URL}/api/sn_chg_rest/v1/change" \
        --request POST \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
        --user "${USERNAME}":"${PASSWORD}" \
        --data "{\"number\":\"${CHANGE_NUMBER}\",\"cmdb_ci\":\"${CMDB_CI}\",\"short_description\":\"${SHORT_DESCRIPTION}\", \"start_date\":\"${ONE_HOUR_AGO_UTC}\",\"end_date\":\"${MORE_THAN_ONE_HOUR_FROM_NOW_UTC}\",\"state\":\"Assess\",\"assignment_group\":\"${SYS_ID_ASSIGNMENT_GROUP}\"}")

    SYS_ID_CHANGE=$(echo "${OUTPUT}" | jq '.result.sys_id.value' | awk -F'"' '{print $2}')
    echo "${SYS_ID_CHANGE}"
}

function approve_change() {
    SYS_ID_CHANGE=$1
    APPROVAL_TYPE=$2

    OUTPUT=$(curl "${URL}/api/now/table/sysapproval_approver?sysapproval=${SYS_ID_CHANGE}&state=requested" \
         --request GET \
         --header "Accept:application/json" \
         --user "${USERNAME}":"${PASSWORD}")
    SYS_ID_APPROVAL=$(echo "${OUTPUT}" | jq ".result[].sys_id" | head -n 1 | awk -F'"' '{print $2}')

    if test "${SYS_ID_APPROVAL}" != ""
    then
        echo "sys_id of first of the approvals from ${APPROVAL_TYPE} is: ${SYS_ID_APPROVAL}"
    else
        echo "Getting approver failed, output of command = ${OUTPUT}"
        exit 1
    fi

    OUTPUT=$(curl "${URL}/api/now/table/sysapproval_approver/${SYS_ID_APPROVAL}" \
        --request PUT \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
        --user "${USERNAME}":"${PASSWORD}" \
        --data "{\"state\":\"approved\"}")
    SYS_ID_APPROVED=$(echo "${OUTPUT}" | jq '.result.sys_id' | awk -F'"' '{print $2}')

    if test "${SYS_ID_APPROVED}" != ""
    then
        echo "sys_id ${SYS_ID_APPROVED} approved"
    else
        echo "Approval failed, output of command = ${OUTPUT}"
        exit 1
    fi
}

function set_change_to_implement() {
    SYS_ID_CHANGE=$1

    OUTPUT=$(curl "${URL}/api/sn_chg_rest/v1/change/${SYS_ID_CHANGE}" \
        --request PATCH \
        --header "Accept:application/json" \
        --header "Content-Type: application/json" \
        --user "${USERNAME}":"${PASSWORD}" \
        --data "{\"state\":\"implement\"}")
    SYS_ID_PATCHED_CHANGE=$(echo "${OUTPUT}" | jq '.result.sys_id.value' | awk -F'"' '{print $2}')

    if test "${SYS_ID_PATCHED_CHANGE}" != ""
    then
        echo "Change ${SYS_ID_PATCHED_CHANGE} set to correct state"
    else
        echo "Changing state failed, output of command = ${OUTPUT}"
        exit 1
    fi
}

check_parameters
SYS_ID_ASSIGNMENT_GROUP=$(find_sys_id_assignment_group_application_development)

RESULT=$(check_change)
if test "${RESULT}" != "${CHANGE_NUMBER}"
then
    echo "Change doesn't exist, adding it..."
else
    echo "Change already exists"
    exit 1
fi

SYS_ID_CHANGE=$(add_change "${SYS_ID_ASSIGNMENT_GROUP}")
if test "${SYS_ID_CHANGE}" != ""
then
    echo "Change added, sys_id = ${SYS_ID_CHANGE}"
else
    echo "Adding change failed"
    exit 1
fi

approve_change "${SYS_ID_CHANGE}" "Application Development"
approve_change "${SYS_ID_CHANGE}" "CAB"
set_change_to_implement "${SYS_ID_CHANGE}"
