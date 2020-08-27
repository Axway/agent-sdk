#!/bin/bash

MODEL_PATH="./pkg/apic/apiserver/models/management/v1alpha1"
COMMENT="// GENERATE: The following code has been modified after code generation"

# for each file that needs changing, you can re-use the following 2 vars if you wish
SEARCH="\s*AutoSubscribe\s*bool.*"
REPLACE="AutoSubscribe bool \`json:\"autoSubscribe\"\`"

######################
# For model_consumer_instance_spec_subscription.go, we want to remove 'omitempty' from AutoSubscribe
######################
# add a comment to the code
sed -i -e "/\s*AutoSubscribe\s*bool.*/i ${COMMENT}" ${MODEL_PATH}/model_consumer_instance_spec_subscription.go
# comment out the line we're changing
sed -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_consumer_instance_spec_subscription.go
# add in the new line we want
sed -i "/AutoSubscribe/a ${REPLACE}" ${MODEL_PATH}/model_consumer_instance_spec_subscription.go
# reformat the code
go fmt ${MODEL_PATH}/model_consumer_instance_spec_subscription.go
