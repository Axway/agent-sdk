#!/bin/bash

MODEL_PATH="${OUTDIR}/models/management/v1alpha1"
COMMENT="// GENERATE: The following code has been modified after code generation"

# for each file that needs changing, you can re-use the following 2 vars if you wish
SEARCH="\s*AutoSubscribe\s*bool.*"
REPLACE="AutoSubscribe bool \`json:\"autoSubscribe\"\`"

# Ubuntu ships with GNU sed, where the suffix for the -i option is optional.
# OS X ships with BSD sed, where the suffix is mandatory, and a backup will be created.
# Using GNU sed prevents the script from breaking, and will not create a backup file.
SED=sed
OS=`uname`
if [[ "$OS" == "Darwin" ]] ; then
    SED=gsed
    type $SED >/dev/null 2>&1 || {
        echo -e >&2 "$SED it not installed. Try: brew install gnu-sed" ;
        exit 1;
    }
fi

######################
# For model_consumer_instance_spec_subscription.go, we want to remove 'omitempty' from AutoSubscribe
######################
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_consumer_instance_spec_subscription.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_consumer_instance_spec_subscription.go
# add in the new line we want
$SED -i "/AutoSubscribe/a ${REPLACE}" ${MODEL_PATH}/model_consumer_instance_spec_subscription.go
# reformat the code
go fmt ${MODEL_PATH}/model_consumer_instance_spec_subscription.go


######################
# For model_consumer_instance_spec.go, we want to turn 	"Icon ConsumerInstanceSpecIcon `json:"icon,omitempty"`" into
# "Icon *ConsumerInstanceSpecIcon `json:"icon,omitempty"`"
######################
SEARCH="\s*Icon\s*ConsumerInstanceSpecIcon.*"
REPLACE="Icon *ConsumerInstanceSpecIcon \`json:\"icon,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_consumer_instance_spec.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_consumer_instance_spec.go
# add in the new line we want
$SED -i "/ConsumerInstanceSpecIcon/a ${REPLACE}" ${MODEL_PATH}/model_consumer_instance_spec.go
# reformat the code
go fmt ${MODEL_PATH}/model_consumer_instance_spec.go

######################
# For model_watch_topic_spec_filters.go.go, we want to turn 	"Scope WatchTopicSpecScope `json:"scope,omitempty"`" into
# "Scope *WatchTopicSpecScope `json:"scope,omitempty"`"
######################
SEARCH="\s*Scope\s*WatchTopicSpecScope.*"
REPLACE="Scope *WatchTopicSpecScope \`json:\"scope,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_watch_topic_spec_filters.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_watch_topic_spec_filters.go
# add in the new line we want
$SED -i "/WatchTopicSpecScope/a ${REPLACE}" ${MODEL_PATH}/model_watch_topic_spec_filters.go
# reformat the code
go fmt ${MODEL_PATH}/model_watch_topic_spec_filters.go

######################
# For AccessRequest.go, we want to turn 	"References AccessRequestReferences `json:"references"`" into
# "References []AccessRequestReferences `json:"references"`"
######################
SEARCH="\s*References\s*interface{}.*"
REPLACE="References []interface{} \`json:\"references\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/AccessRequest.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/AccessRequest.go
# add in the new line we want
$SED -i "/References\s/a ${REPLACE}" ${MODEL_PATH}/AccessRequest.go
# reformat the code
go fmt ${MODEL_PATH}/AccessRequest.go


######################
# Update any time imports in the models, we want to turn "time" into
# time "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
######################
MODELS=`find ${OUTDIR}/models -type f -name "model_*.go"`

SEARCH="\s*\"time\"$"
REPLACE="time \"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1\""
for file in ${MODELS}; do
    if grep -e ${SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${SEARCH}/i ${COMMENT}" ${file}
        # comment out the line we're changing
        $SED -i -e "s/${SEARCH}/\/\/ &/" ${file}
        # add in the new line we want
        $SED -i "/\/\/${SEARCH}/a ${REPLACE}" ${file}
        # reformat the code
        go fmt ${file}
    fi
done


######################
# Update any OneOf types to be interface{}
######################
MODELS=`find ${OUTDIR}/models -type f -name "model_*.go"`

SEARCH="OneOf.*\s"
REPLACE="interface{} "
for file in ${MODELS}; do
    if grep -e ${SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${SEARCH}/i ${COMMENT}" ${file}
        # replace the Oneof type
        $SED -i -e "s/${SEARCH}/${REPLACE}/g" ${file}
        # reformat the code
        go fmt ${file}
    fi
done

######################
# Remove the ManagementV1alpha1 prefix from the resources generated
######################
MODELS=`find ${OUTDIR}/models -type f -name "model_*.go"`

SEARCH="ManagementV1alpha1"
REPLACE=""
for file in ${MODELS}; do
    if grep -e ${SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${SEARCH}/i ${COMMENT}" ${file}
        # remove the prefix
        $SED -i -e "s/${SEARCH}/${REPLACE}/g" ${file}
        # reformat the code
        go fmt ${file}
    fi
done

######################
# Remove the CatalogV1alpha1 prefix from the resources generated
######################
MODELS=`find ${OUTDIR}/models -type f -name "model_*.go"`

SEARCH="CatalogV1alpha1"
REPLACE=""
for file in ${MODELS}; do
    if grep -e ${SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${SEARCH}/i ${COMMENT}" ${file}
        # remove the prefix
        $SED -i -e "s/${SEARCH}/${REPLACE}/g" ${file}

        # reformat the code
        go fmt ${file}
    fi
done

######################
# Update the following STATES to include the type infront of the constant
######################
MODELS=`find ${OUTDIR}/models -type f -name "model_*_state.go"`
STATES="DRAFT ACTIVE DEPRECATED ARCHIVED"

for file in ${MODELS}; do
    stateType=`grep "List of" ${file} | awk '{print $4}'`
    for state in ${STATES}; do
        if grep -e ${state} ${file} >> /dev/null; then
            # add a comment to the code
            $SED -i -e "/${state}/i ${COMMENT}" ${file}
            # replace the state
            $SED -i -e "s/${state}/${stateType}${state}/g" ${file}
            # reformat the code
            go fmt ${file}
        fi
    done
done


######################
# Update the Status subresource in generated model to use v1.ResourceStatus
######################
MODELS=`find ${OUTDIR}/models -type f -name "*.go" \
    ! -name 'model_*.go' \
    ! -name 'AmplifyRuntimeConfig.go' \
    ! -name 'AssetMapping.go' \
    ! -name 'ConsumerInstance.go' \
    ! -name 'DiscoveryAgent.go' \
    ! -name 'GovernanceAgent.go' \
    ! -name 'TraceabilityAgent.go'`

SEARCH="\s*Status.*\s*\`json:\"status\"\`$"
REPLACE="Status *apiv1.ResourceStatus \`json:\"status\"\`"
SEARCH_UNMARSHAL="\s*err\s=\sjson\.Unmarshal(sr,\s\&res.Status)$"
REPLACE_UNMARSHAL="res.Status = &apiv1.ResourceStatus{}\nerr = json.Unmarshal(sr, res.Status)"
for file in ${MODELS}; do
    if grep -e ${SEARCH} ${file} >> /dev/null; then
        # comment out the line we're changing
        $SED -i -e "s/${SEARCH}/\/\/ &/" ${file}
        # add in the new line we want
        $SED -i "/\/\/${SEARCH}/a ${REPLACE}" ${file}
        # Update the unmarshal call
        $SED -i -e "s/${SEARCH_UNMARSHAL}/\/\/ &/" ${file}
        # add in the new line we want
        $SED -i "/\/\/${SEARCH_UNMARSHAL}/a ${REPLACE_UNMARSHAL}" ${file}
        # reformat the code
        go fmt ${file}
    fi
done

######################
# Update the following REQUEST states to include the type infront of the constant
######################
# MODELS=`find ${OUTDIR}/models -type f -name "model_*_request.go"`
# REQUESTS="PROVISION RENEW"

# for file in ${MODELS}; do
#     requestType=`grep "List of" ${file} | awk '{print $4}'`
#     for request in ${REQUESTS}; do
#         if grep -e ${request} ${file} >> /dev/null; then
#             # add a comment to the code
#             $SED -i -e "/${request}/i ${COMMENT}" ${file}
#             # replace the state
#             $SED -i -e "s/${request}/${requestType}${request}/g" ${file}
#             # reformat the code
#             go fmt ${file}
#         fi
#     done
# done

######################
# Update any OneOf types to be interface{}
######################
MODELS=`find ${OUTDIR}/models -type f -name "model_*.go"`

SEARCH="float32.*\s"
REPLACE="float64 "
for file in ${MODELS}; do
    if grep -e ${SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${SEARCH}/i ${COMMENT}" ${file}
        # replace the float32 type
        $SED -i -e "s/${SEARCH}/${REPLACE}/g" ${file}
        # reformat the code
        go fmt ${file}
    fi
done


######################
# Change Endpoint details to map[string]interface{}
######################
MODELS=`find ${OUTDIR}/models -type f -name "model_api_service_instance_spec_routing.go"`

SEARCH="\s*Details.*$"
REPLACE="Details map[string]interface{} \`json:\"details,omitempty\"\`"
for file in ${MODELS}; do
    if grep -e ${SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${SEARCH}/i ${COMMENT}" ${file}
        # replace the float32 type
        $SED -i -e "s/${SEARCH}/${REPLACE}/g" ${file}
        # reformat the code
        go fmt ${file}
    fi
done