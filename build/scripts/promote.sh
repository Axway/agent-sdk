#!/bin/bash

check_required_param() {
    echo $1
    if [ -z $1 ]; then
        return 1
    fi

    pat='refs/tags/v[0-9].[0-9].[0-9]'
    if [[ $1 =~ $pat ]]; then
        return 0
    fi
    return 1
}

set_version_variables() {
    var1=$(echo $1 | cut -f3 -d/)
    version="${var1:1}"
    export VERSION=$version
    export BASE_DIR=$(realpath $(dirname $0)/../..)
    # export MAJOR_VERSION=$(echo $version | cut -d. -f1)
    # export MINOR_VERSION=$(echo $version | cut -d. -f2)
    # export PATCH_VERSION=$(echo $version | cut -d. -f3)
}

# tag_branch() {
#     git tag -a v${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION} master -m "Version ${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}"
#     git push origin v${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}
# }

# update_helm_app_version() {   
#     echo "Updating appVersion for helm chart." 
#     sed -i "/appVersion:.*/s/.*/appVersion: \"${VERSION}\"/" helm/v7-discovery/Chart.yaml
# }

# promote_version() {
#     if [ "${PROMOTION_TYPE}" == "patch" ]; then
#         let PATCH_VERSION=($PATCH_VERSION+1)
#     fi

#     if [ "${PROMOTION_TYPE}" == "minor" ]; then
#         let MINOR_VERSION=($MINOR_VERSION+1)
#         PATCH_VERSION=0
#     fi

#     if [ "${PROMOTION_TYPE}" == "major" ]; then
#         let MAJOR_VERSION=($MAJOR_VERSION+1)
#         MINOR_VERSION=0
#         PATCH_VERSION=0
#     fi

#     export MAJOR_VERSION
#     export MINOR_VERSION
#     export MINOR_VERSION
#     export PATCH_VERSION
    
#     export NEW_VERSION="${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}"
    # export MSG="Switching to new version: ${VERSION}"
# }
#     git checkout ${CI_COMMIT_REF_NAME}

update_version_file() {
    echo ${VERSION} > ${BASE_DIR}/version
}

# update_dependencies_to_master() {
#     cd ${BASE_DIR}

#     make dep-sdk
# }

commit_promotion() {
    echo ${MSG}
    cd ${BASE_DIR}
    git status
    git add .

    # git commit -m "INT - ${MSG}"
    # git push origin master
}

main() {
    check_required_param $1
    if [ $? -eq 1 ]; then
        echo "Promotion of release not completed. Missing parameter for release version (e.g. refs/tags/v1.2.3)"
        echo "Skipping Promotion."
        exit
    fi

    set_version_variables $1
    # git checkout ${CI_COMMIT_REF_NAME}

    # if [ $CHECK_ONLY -eq 1 ]; then
    #     echo "Check completed"
    #     exit
    # fi

    # update_helm_app_version
    # echo "Creating tag for new release ${VERSION}"
    # tag_branch

    # promote_version
    # echo $MSG

    # Update versions file
    echo "Updating version file"
    update_version_file
    # update_dependencies_to_master

    # # Commit file
    echo "Committing the new promoted version to main"
    commit_promotion
}

main $@
