#!/bin/bash

check_required_param() {
    echo $1
    if [ -z $1 ]; then
        return 1
    fi

    pat='v[0-9]+\.[0-9]+\.[0-9]+'
    if [[ $1 =~ $pat ]]; then
        return 0
    fi
    return 1
}

set_version_variables() {
    version="${1:1}"

    let MAJOR_VERSION=$(echo $version | cut -d. -f1)
    let MINOR_VERSION=$(echo $version | cut -d. -f2)
    let PATCH_VERSION=$(echo $version | cut -d. -f3)
    let NEW_PATCH_VERSION=($PATCH_VERSION+1)

    # right now, this only does patch versioning.
    export NEW_VERSION="${MAJOR_VERSION}.${MINOR_VERSION}.${NEW_PATCH_VERSION}"

    export BASE_DIR=$(realpath $(dirname $0)/../..)
    export MSG="update to new release ${NEW_VERSION}"
}

checkout_main() {
    # need these in order to commit
    git config --global user.name "Gitlab action"
    git config --global user.email "gitaction@axway.com"
    git fetch
    git checkout main
}

update_version_file() {
    echo "Updating version file to version ${NEW_VERSION}"
    echo ${NEW_VERSION} > ${BASE_DIR}/version
}

commit_promotion() {
    echo "Committing the new promoted version to main"
    cd ${BASE_DIR}
    git add version
    git commit -m "INT - ${MSG}"
    git push origin main
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

    update_version_file

    commit_promotion
}

main $@
