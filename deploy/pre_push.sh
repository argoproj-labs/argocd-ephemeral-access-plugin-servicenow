#!/bin/bash

function replace_servicenow_url() {
    echo "Replace ServiceNow URL..."

    ./replace_servicenow_url.sh https://dev123456.service-now.com
    if test $? -ne 0
    then
        exit 1
    fi
}

function markdown_lint() {
    echo "Markdown lint..."

    markdownlint .
    if test $? -ne 0
    then
        exit 1
    fi
}

function shell_check() {
    echo "Shell check..."

    find . -name "*.sh" -exec shellcheck {} \;
    if test $? -ne 0
    then
        exit 1
    fi
}

function cfn_lint_cloudformation_file() {
    CLOUDFORMATION_FILE=$1

    cfn-lint "${CLOUDFORMATION_FILE}"
    if test $? -ne 0
    then
        exit 1
    fi
}

function cfn_lint() {
    echo "cfn-lint..."
    cfn_lint_cloudformation_file ./examples/aws/cloudformation.yaml
    cfn_lint_cloudformation_file ./examples/dev-test/aws-dev/cloudformation.yaml
    cfn_lint_cloudformation_file ./examples/dev-test/aws-pre-install-plugin/cloudformation.yaml
}

function go_lint() {
    echo "Go lint..."

    golangci-lint run
    if test $? -ne 0
    then
        exit 1
    fi
}

function unit_test() {
    echo "Unit tests..."

    go test ./... -v -coverprofile=coverage
    if test $? -ne 0
    then
        exit 1
    fi
}

function unix_to_dos() {
    echo "Unix to dos..."

    find . -name "*.sh" -exec unix2dos {} \;
}


replace_servicenow_url

cd ..

markdown_lint
shell_check
cfn_lint
go_lint
unit_test
unix_to_dos

cd - || exit
