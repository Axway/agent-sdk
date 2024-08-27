#!/bin/bash

#
# bump the SDK's major version
#
# usage: ./bump_major_version
#
# rules:
#   - update 'module' line (line 1) in go.mod
#       (e.g. module github.com/Axway/agent-sdk => module github.com/Axway/agent-sdk/v3
#   - update all .go file imports that reference the sdk
#       (e.g. "github.com/Axway/agent-sdk/pkg/agent => module github.com/Axway/agent-sdk/v3/pkg/agent
#   - note that going from v1 to v2 requires different changes, as there is no 'v1' in the path
#       (e.g. module github.com/Axway/agent-sdk/pkg/agent => module github.com/Axway/agent-sdk/pkg/agent

check_required_variables() {
    set -x
    if [[ -z ${SDK_VERSION} ]]; then
        echo "Usage: ./bump_major_version.sh"
    fi
}

# set up the actual repo name
set_repo_info() {
    export REPO_NAME="agent-sdk"

    SDK_PREV_VERSION=$(($SDK_VERSION-1))
    if [[ ${SDK_VERSION} == "2" ]]; then
        SDK_PREV_VERSION=""
    fi

    NEW_SDK_PATH="github.com\/Axway\/agent-sdk\/v${SDK_VERSION}"
    OLD_SDK_PATH="github.com\/Axway\/agent-sdk"
    if [[ ${SDK_PREV_VERSION} != "" ]]; then
        OLD_SDK_PATH+="\/v${SDK_PREV_VERSION}"
    fi
}

checkout_repo() {
    echo "Checking out the repo for ${REPO_NAME}"

    git checkout -b APIGOV-bumpVersion
}

commit_and_push_updates() {
    echo -e "Committing the updates in ${REPO_NAME}"

    // first, remove any backup files created by sed on a mac
    find . -type f -name '*.*-e' -delete

    git status
    git add .
    git fetch

    git commit -m "Updating major version"
    git push -u origin APIGOV-bumpVersion
    git checkout master
    git branch -d APIGOV-bumpVersion
}

# checkout the repo, update the version dependencies, and commit
update_component() {
    set_repo_info
    checkout_repo
    update_gomod
    update_imports
    commit_and_push_updates
}

update_gomod() {
    echo -e "Updating go.mod for ${REPO_NAME}"

    # update ref to the SDK
    sed -i -e "s/${OLD_SDK_PATH}/${NEW_SDK_PATH}/" go.mod

    # remove all of the retract lines at the end
    sed -i -e "/^retract (/q; /^retract/d" go.mod
}

update_imports() {
    echo -i -e "Updating the imports for the .go files"

    find . -type f -name "*.go" -exec sed -i -e "s/${OLD_SDK_PATH}/${NEW_SDK_PATH}/g" {} \;
    sed -i -e "s/${OLD_SDK_PATH}/${NEW_SDK_PATH}/g" .csr-profile.json

    go mod tidy
}

main() {
    read -p "Enter the major version # for of the SDK: " SDK_VERSION

    check_required_variables

    cd ..

    update_component

    exit 0
}

main $@

