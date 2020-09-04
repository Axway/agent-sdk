#!/bin/bash

MODEL_PATH="./pkg/apic/apiserver/models/management/v1alpha1"
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
