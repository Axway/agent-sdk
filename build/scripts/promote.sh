#!/bin/bash

check_required_env_variable() {
    if [ -z ${PROMOTION_TYPE} ]; then
        return 1
    fi

    if [ "${CI_COMMIT_REF_NAME}" != "master" ]; then
        return 1
    fi
    return 0
}

set_version_variables() {
    export BASE_DIR=$(realpath $(dirname $0)/../..)
    export VERSION=$(cat ${BASE_DIR}/version)
    echo "Current version is $VERSION"
    export MAJOR_VERSION=$(echo $VERSION | cut -d. -f1)
    export MINOR_VERSION=$(echo $VERSION | cut -d. -f2)
    export PATCH_VERSION=$(echo $VERSION | cut -d. -f3)
}

tag_branch() {
    git tag -a ${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION} master -m "Version ${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}"
    git push origin ${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}
}

promote_version() {
    if [ "${PROMOTION_TYPE}" == "patch" ]; then
        let PATCH_VERSION=($PATCH_VERSION+1)
    fi

    if [ "${PROMOTION_TYPE}" == "minor" ]; then
        let MINOR_VERSION=($MINOR_VERSION+1)
        PATCH_VERSION=0
    fi

    if [ "${PROMOTION_TYPE}" == "major" ]; then
        let MAJOR_VERSION=($MAJOR_VERSION+1)
        MINOR_VERSION=0
        PATCH_VERSION=0
    fi

    export MAJOR_VERSION
    export MINOR_VERSION
    export MINOR_VERSION
    export PATCH_VERSION
    
    export NEW_VERSION="${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}"
    export MSG="Switching to new version: ${NEW_VERSION}"
    git checkout ${CI_COMMIT_REF_NAME}
}

update_version_file() {
    echo ${NEW_VERSION} > ${BASE_DIR}/version
}

commit_promotion() {
    cd ${BASE_DIR}
    git status
    git add ${BASE_DIR}/version

    git commit -m "INT - ${MSG}"
    git push origin master
}

main() {
    CHECK_ONLY=0
    case $1 in
      check)
        CHECK_ONLY=1
        ;;
    esac

    check_required_env_variable
    if [ $? -eq 1 ]; then
        echo "Build not started for promotion. Missing environment variable PROMOTION_TYPE or CI_COMMIT_REF_NAME is not 'master'."
        echo "Skipping Promotion."
        exit
    fi

    set_version_variables
    git checkout ${CI_COMMIT_REF_NAME}

    if [ $CHECK_ONLY -eq 1 ]; then
        echo "Check completed"
        exit
    fi

    echo "Creating tag for new release ${VERSION}"
    tag_branch

    promote_version
    echo $MSG

    # Update versions file
    echo "Updating version file"
    update_version_file

    # Commit file
    echo "Committing the new promoted version to master"
    commit_promotion
}

main $@
