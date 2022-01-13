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
            # replace the Oneof type
            $SED -i -e "s/${state}/${stateType}${state}/g" ${file}
            # reformat the code
            go fmt ${file}
        fi
    done
done