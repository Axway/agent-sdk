#!/bin/bash

check_required_param() {
    echo $1
    if [ -z $1 ]; then
        return 1
    fi

    # pat='refs/tags/v[0-9].[0-9].[0-9]'
    pat='v[0-9].[0-9].[0-9]'
    if [[ $1 =~ $pat ]]; then
        return 0
    fi
    return 1
}

set_version_variables() {
    # var1=$(echo $1 | cut -f3 -d/)
    version="${1:1}"
    export VERSION=$version
    export BASE_DIR=$(realpath $(dirname $0)/../..)
    export MSG="update to new release ${VERSION}"
}

checkout_main() {
    git config --global user.name "Gitlab action"
    git config --global user.email "gitaction@axway.com"
    git fetch
    git checkout APIGOV-12345
}

update_version_file() {
    echo "Updating version file"
    echo ${VERSION} > ${BASE_DIR}/version
}

commit_promotion() {
    echo "Committing the new promoted version to main"
    # echo ${MSG}
    cd ${BASE_DIR}
    # need these in order to commit
    # git config --global user.name "Gitlab action"
    # git config --global user.email "gitaction@axway.com"
    # git status
    git add version
    git commit -m "INT - ${MSG}"
    git push origin APIGOV-12345
}

main() {
    check_required_param $1
    if [ $? -eq 1 ]; then
        echo "Promotion of release not completed. Missing parameter for release version (e.g. v1.2.3)"
        echo "version file not updated. You can update it manually if you wish."
        exit
    fi

    checkout_main

    set_version_variables $1

    # Update versions file
    update_version_file

    # # Commit file
    commit_promotion
}

main $@
