#!/bin/bash

REPLACE_BY=$1

if test "${REPLACE_BY}" == ""
then
	echo "Usage  : replace-servicenow-url.sh URL"
	echo "Example: replace-servicenow-url.sh https://dev123456.service-now.com"
	exit 1
fi

find ../* -type f -name "*.sh" -exec sed -i "s^https://dev[0-9]*.service-now.com^${REPLACE_BY}^" {} \;
