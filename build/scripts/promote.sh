#!/bin/bash
check_required_param() {
    if [ -z $1 ]; then
        return 1
    fi

    # version must be of the form: x.y.z
    pat='[0-9]+\.[0-9]+\.[0-9]+'
    if [[ $1 =~ $pat ]]; then
        return 0
    fi
    return 1
}

set_version_variables() {
    export NEW_VERSION=$1
    export BASE_DIR=$(realpath $(dirname $0)/../..)
    export MSG="update to new release ${NEW_VERSION}"
}

update_version_file() {
    echo "Updating version file to version ${NEW_VERSION}"
    echo ${NEW_VERSION} > ${BASE_DIR}/version
}

main() {
    check_required_param $1
    if [ $? -eq 1 ]; then
        echo "Promotion of release not completed. Missing parameter for release version (e.g. v1.2.3)"
        echo "version file not updated. You can update it manually if you wish."
        exit
    fi
    
    set_version_variables $1

    update_version_file
}

main $@
