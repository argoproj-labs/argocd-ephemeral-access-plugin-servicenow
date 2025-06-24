#!/bin/bash
shopt -s lastpipe

PROGRAM_FILE="main.go"
TEST_FILE="main_test.go"
ERROR_FOUND=0

grep "func" ${PROGRAM_FILE} | grep -v main\( | awk '{print $4}'| awk -F'(' '{print $1}' | while IFS= read -r FUNCTION_NAME
do
	grep -i "test${FUNCTION_NAME}" "${TEST_FILE}" > /dev/null
	if test $? == 1
	then
		echo "Function ${FUNCTION_NAME} doesn't have tests"
		ERROR_FOUND=1
	fi
done

echo ""
if test ${ERROR_FOUND} -eq 0
then
	echo "All functions in ${PROGRAM_FILE} have tests in ${TEST_FILE}"
else
	echo "Errors found, please fix them"
	exit 1
fi
