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
# For model_traceability_agent_agentstate.go, we want to turn 	"Sampling TraceabilityAgentAgentstateSampling `json:"sampling,omitempty"`" into
# "Sampling *TraceabilityAgentAgentstateSampling `json:"sampling,omitempty"`"
######################
SEARCH="\s*Sampling\s*TraceabilityAgentAgentstateSampling.*"
REPLACE="Sampling *TraceabilityAgentAgentstateSampling \`json:\"sampling,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_traceability_agent_agentstate.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_traceability_agent_agentstate.go
# add in the new line we want
$SED -i "/TraceabilityAgentAgentstateSampling/a ${REPLACE}" ${MODEL_PATH}/model_traceability_agent_agentstate.go
# reformat the code
go fmt ${MODEL_PATH}/model_traceability_agent_agentstate.go

######################
# For model_watch_topic_spec_filters.go, we want to turn 	"Scope WatchTopicSpecScope `json:"scope,omitempty"`" into
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
# For APIService.go, we want to turn    "Compliance ApiServiceCompliance `json:"compliance"`" into
# "Compliance *ApiServiceCompliance `json:"compliance,omitempty"`"
######################
SEARCH="\s*Compliance\s*ApiServiceCompliance.*"
REPLACE="Compliance *ApiServiceCompliance \`json:\"compliance,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIService.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIService.go
# add in the new line we want
$SED -i "/ApiServiceCompliance\s/a ${REPLACE}" ${MODEL_PATH}/APIService.go

######################
# For APIService.go, we want to turn    Agentdetails ApiServiceAgentdetails `json:"agentdetails"` into
# "Agentdetails *ApiServiceAgentdetails `json:"agentdetails,omitempty"`"
######################
SEARCH="\s*Agentdetails\s*ApiServiceAgentdetails.*"
REPLACE="Agentdetails *ApiServiceAgentdetails \`json:\"agentdetails,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIService.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIService.go
# add in the new line we want
$SED -i "/ApiServiceAgentdetails\s/a ${REPLACE}" ${MODEL_PATH}/APIService.go
# reformat the code
go fmt ${MODEL_PATH}/APIService.go


######################
# For APIService.go, we want to turn    "Source ApiServiceSource `json:"source"`" into
# "Source *ApiServiceSource `json:"source"`"
######################
SEARCH="\s*Source\s*ApiServiceSource.*"
REPLACE="Source *ApiServiceSource \`json:\"source,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIService.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIService.go
# add in the new line we want
$SED -i "/ApiServiceSource\s/a ${REPLACE}" ${MODEL_PATH}/APIService.go
go fmt ${MODEL_PATH}/APIService.go

######################
# For APIService.go, we want to turn    "Agentdetails ApiServiceAgentdetails `json:"agentdetails"`" into
# "Agentdetails *ApiServiceAgentdetails `json:"agentdetails"`"
######################
SEARCH="\s*Agentdetails\s*ApiServiceAgentdetails.*"
REPLACE="Agentdetails *ApiServiceAgentdetails \`json:\"agentdetails,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIService.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIService.go
# add in the new line we want
$SED -i "/ApiServiceAgentdetails\s/a ${REPLACE}" ${MODEL_PATH}/APIService.go
go fmt ${MODEL_PATH}/APIService.go

######################
# For APIService.go, we want to turn    "Profile ApiServiceProfile `json:"profile"`" into
# "Profile *ApiServiceProfile `json:"profile"`"
######################
SEARCH="\s*Profile\s*ApiServiceProfile.*"
REPLACE="Profile *ApiServiceProfile \`json:\"profile,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIService.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIService.go
# add in the new line we want
$SED -i "/ApiServiceProfile\s/a ${REPLACE}" ${MODEL_PATH}/APIService.go
go fmt ${MODEL_PATH}/APIService.go

######################
# For APIService.go, we want to turn    "Appinfo ApiServiceAppinfo `json:"appinfo"`" into
# "Appinfo *ApiServiceAppinfo `json:"appinfo"`"
######################
SEARCH="\s*Appinfo\s*ApiServiceAppinfo.*"
REPLACE="Appinfo *ApiServiceAppinfo \`json:\"appinfo,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIService.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIService.go
# add in the new line we want
$SED -i "/ApiServiceAppinfo\s/a ${REPLACE}" ${MODEL_PATH}/APIService.go
# reformat the code
go fmt ${MODEL_PATH}/APIService.go


######################
# For model_api_service_source.go, we want to turn "DataplaneType ApiServiceSourceDataplaneType `json:"dataplaneType,omitempty"`" into
# DataplaneType *ApiServiceSourceDataplaneType `json:"dataplaneType,omitempty"`"
######################
SEARCH="\s*DataplaneType\s*ApiServiceSourceDataplaneType.*"
REPLACE="DataplaneType *ApiServiceSourceDataplaneType \`json:\"dataplaneType,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_api_service_source.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_api_service_source.go
# add in the new line we want
$SED -i "/ApiServiceSourceDataplaneType\s/a ${REPLACE}" ${MODEL_PATH}/model_api_service_source.go
# reformat the code
go fmt ${MODEL_PATH}/model_api_service_source.go

######################
# For model_api_service_source.go, we want to turn "References ApiServiceSourceReferences `json:"references,omitempty"`" into
# References *ApiServiceSourceReferences `json:"references,omitempty"`"
######################
SEARCH="\s*References\s*ApiServiceSourceReferences.*"
REPLACE="References *ApiServiceSourceReferences \`json:\"references,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_api_service_source.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_api_service_source.go
# add in the new line we want
$SED -i "/ApiServiceSourceReferences\s/a ${REPLACE}" ${MODEL_PATH}/model_api_service_source.go
# reformat the code
go fmt ${MODEL_PATH}/model_api_service_source.go


######################
# For APIServiceInstance.go, we want to turn    "Compliance ApiServiceInstanceCompliance `json:"compliance"`" into
# "Compliance *ApiServiceInstanceCompliance `json:"compliance,omitempty"`"
######################
SEARCH="\s*Compliance\s*ApiServiceInstanceCompliance.*"
REPLACE="Compliance *ApiServiceInstanceCompliance \`json:\"compliance,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIServiceInstance.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIServiceInstance.go
# add in the new line we want
$SED -i "/ApiServiceInstanceCompliance\s/a ${REPLACE}" ${MODEL_PATH}/APIServiceInstance.go
# reformat the code
go fmt ${MODEL_PATH}/APIServiceInstance.go

######################
# For APIServiceInstance.go, we want to turn    "Lifecycle  ApiServiceInstanceLifecycle   `json:"lifecycle"`" into
# "Lifecycle  *ApiServiceInstanceLifecycle   `json:"lifecycle,omitempty"`"
######################
SEARCH="\s*Lifecycle\s*ApiServiceInstanceLifecycle.*"
REPLACE="Lifecycle *ApiServiceInstanceLifecycle \`json:\"lifecycle,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIServiceInstance.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIServiceInstance.go
# add in the new line we want
$SED -i "/ApiServiceInstanceLifecycle\s/a ${REPLACE}" ${MODEL_PATH}/APIServiceInstance.go
# reformat the code
go fmt ${MODEL_PATH}/APIServiceInstance.go

######################
# For APIServiceInstance.go, we want to turn    "Source ApiServiceInstanceSource `json:"source"`" into
# "Source *ApiServiceInstanceSource `json:"source"`"
######################
SEARCH="\s*Source\s*ApiServiceInstanceSource.*"
REPLACE="Source *ApiServiceInstanceSource \`json:\"source,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIServiceInstance.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIServiceInstance.go
# add in the new line we want
$SED -i "/ApiServiceInstanceSource\s/a ${REPLACE}" ${MODEL_PATH}/APIServiceInstance.go
# reformat the code
go fmt ${MODEL_PATH}/APIServiceInstance.go

######################
# For APIServiceInstance.go, we want to turn    "Traceable  ApiServiceInstanceTraceable   `json:"traceable"`" into
# "Traceable  *ApiServiceInstanceTraceable   `json:"traceable,omitempty"`"
######################
SEARCH="\s*Traceable\s*ApiServiceInstanceTraceable.*"
REPLACE="Traceable *ApiServiceInstanceTraceable \`json:\"traceable,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIServiceInstance.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIServiceInstance.go
# add in the new line we want
$SED -i "/ApiServiceInstanceTraceable\s/a ${REPLACE}" ${MODEL_PATH}/APIServiceInstance.go
# reformat the code
go fmt ${MODEL_PATH}/APIServiceInstance.go

######################
# For APIServiceInstance.go, we want to turn    "Sampletrigger ApiServiceInstanceSampletrigger `json:"sampletrigger"`" into
# "Sampletrigger *ApiServiceInstanceSampletrigger `json:"sampletrigger,omitempty"`"
######################
SEARCH="\s*Sampletrigger\s*ApiServiceInstanceSampletrigger.*"
REPLACE="Sampletrigger *ApiServiceInstanceSampletrigger \`json:\"sampletrigger,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIServiceInstance.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIServiceInstance.go
# add in the new line we want
$SED -i "/ApiServiceInstanceSampletrigger\s/a ${REPLACE}" ${MODEL_PATH}/APIServiceInstance.go
# reformat the code
go fmt ${MODEL_PATH}/APIServiceInstance.go

######################
# For APIServiceInstance.go, we want to turn    "Samplestate   ApiServiceInstanceSamplestate   `json:"samplestate"`" into
# "Samplestate   *ApiServiceInstanceSamplestate   `json:"samplestate,omitempty"`"
######################
SEARCH="\s*Samplestate\s*ApiServiceInstanceSamplestate.*"
REPLACE="Samplestate *ApiServiceInstanceSamplestate \`json:\"samplestate,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/APIServiceInstance.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/APIServiceInstance.go
# add in the new line we want
$SED -i "/ApiServiceInstanceSamplestate\s/a ${REPLACE}" ${MODEL_PATH}/APIServiceInstance.go
# reformat the code
go fmt ${MODEL_PATH}/APIServiceInstance.go

######################
# For model_api_service_instance_spec.go, we want to turn "Mock ApiServiceInstanceSpecMock `json:"mock,omitempty"`" into
# Mock     *ApiServiceInstanceSpecMock `json:"mock,omitempty"`"
######################
SEARCH="\s*Mock\s*ApiServiceInstanceSpecMock.*"
REPLACE="Mock *ApiServiceInstanceSpecMock \`json:\"mock,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_api_service_instance_spec.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_api_service_instance_spec.go
# add in the new line we want
$SED -i "/ApiServiceInstanceSpecMock\s/a ${REPLACE}" ${MODEL_PATH}/model_api_service_instance_spec.go
# reformat the code
go fmt ${MODEL_PATH}/model_api_service_instance_spec.go

######################
# For model_api_service_instance_source.go, we want to turn "DataplaneType ApiServiceInstanceSourceDataplaneType `json:"dataplaneType,omitempty"`" into
# DataplaneType *ApiServiceInstanceSourceDataplaneType `json:"dataplaneType,omitempty"`"
######################
SEARCH="\s*DataplaneType\s*ApiServiceInstanceSourceDataplaneType.*"
REPLACE="DataplaneType *ApiServiceInstanceSourceDataplaneType \`json:\"dataplaneType,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_api_service_instance_source.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_api_service_instance_source.go
# add in the new line we want
$SED -i "/ApiServiceInstanceSourceDataplaneType\s/a ${REPLACE}" ${MODEL_PATH}/model_api_service_instance_source.go
# reformat the code
go fmt ${MODEL_PATH}/model_api_service_instance_source.go

######################
# For model_api_service_instance_source.go, we want to turn "References ApiServiceInstanceSourceReferences `json:"references,omitempty"`" into
# References *ApiServiceInstanceSourceReferences `json:"references,omitempty"`"
######################
SEARCH="\s*References\s*ApiServiceInstanceSourceReferences.*"
REPLACE="References *ApiServiceInstanceSourceReferences \`json:\"references,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_api_service_instance_source.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_api_service_instance_source.go
# add in the new line we want
$SED -i "/ApiServiceInstanceSourceReferences\s/a ${REPLACE}" ${MODEL_PATH}/model_api_service_instance_source.go
# reformat the code
go fmt ${MODEL_PATH}/model_api_service_instance_source.go

######################
# For model_api_service_instance_source.go, we want to turn "Compliance *ApiServiceInstanceSourceCompliance `json:"compliance,omitempty"`" into
# Compliance *ApiServiceInstanceSourceCompliance `json:"compliance,omitempty"`"
######################
SEARCH="\s*Compliance\s*ApiServiceInstanceSourceCompliance.*"
REPLACE="Compliance *ApiServiceInstanceSourceCompliance \`json:\"compliance,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_api_service_instance_source.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_api_service_instance_source.go
# add in the new line we want
$SED -i "/ApiServiceInstanceSourceCompliance\s/a ${REPLACE}" ${MODEL_PATH}/model_api_service_instance_source.go
# reformat the code
go fmt ${MODEL_PATH}/model_api_service_instance_source.go

######################
# For model_traceability_agent_spec_config.go, we want to turn 	"Traceable TraceabilityAgentSpecConfigSampling  `json:"sampling,omitempty"`" into
# "Traceable  *TraceabilityAgentSpecConfigSampling  `json:"sampling"`"
######################
SEARCH="\s*Sampling\s*TraceabilityAgentSpecConfigSampling.*"
REPLACE="Sampling *TraceabilityAgentSpecConfigSampling \`json:\"sampling,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/model_traceability_agent_spec_config.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/model_traceability_agent_spec_config.go
# add in the new line we want
$SED -i "/TraceabilityAgentSpecConfigSampling/a ${REPLACE}" ${MODEL_PATH}/model_traceability_agent_spec_config.go
# reformat the code
go fmt ${MODEL_PATH}/model_traceability_agent_spec_config.go


######################
# For Environment.go, we want to turn    "Traceable  EnvironmentTraceable   `json:"traceable"`" into
# "Traceable  *EnvironmentTraceable   `json:"traceable,omitempty"`"
######################
SEARCH="\s*Traceable\s*EnvironmentTraceable.*"
REPLACE="Traceable *EnvironmentTraceable \`json:\"traceable,omitempty\"\`"
# add a comment to the code
$SED -i -e "/${SEARCH}/i ${COMMENT}" ${MODEL_PATH}/Environment.go
# comment out the line we're changing
$SED -i -e "s/${SEARCH}/\/\/ &/" ${MODEL_PATH}/Environment.go
# add in the new line we want
$SED -i "/EnvironmentTraceable\s/a ${REPLACE}" ${MODEL_PATH}/Environment.go
# reformat the code
go fmt ${MODEL_PATH}/Environment.go

######################
# Update the Status subresource in generated model to use v1.ResourceStatus
######################
MODELS=`find ${OUTDIR}/models -type f -name "*.go" \
    ! -name 'model_*.go' \
    ! -name 'AmplifyRuntimeConfig.go' \
    ! -name 'AssetMapping.go' \
    ! -name 'DiscoveryAgent.go' \
    ! -name 'TraceabilityAgent.go' \
    ! -name 'ComplianceAgent.go'`

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
# Update the following STATES to include the type infront of the constant
######################
STATES="DRAFT ACTIVE DEPRECATED ARCHIVED ARCHIVING"
MODELS=`find ${OUTDIR}/models -type f -name "model_*_state.go"`

for file in ${MODELS}; do
    stateType=`grep "List of" ${file} | awk '{print $4}'`
    for state in ${STATES}; do
        if grep -e ${state} ${file} >> /dev/null; then
            # add a comment to the code
            $SED -i -e "/${state}/i ${COMMENT}" ${file}
            # replace the state
            $SED -i -e "s/${state}/${stateType}${state}/g" ${file}
        fi
    done
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
    fi
done


MODELS=`find ${OUTDIR}/models -type f -name "model_*.go"`

######################
# Update any time imports in the models, we want to turn "time" into
# time "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
######################
TIME_SEARCH="\s*\"time\"$"
TIME_REPLACE="time \"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1\""

######################
# Update any OneOf types to be interface{}
######################
ONEOF_SEARCH="OneOf.*\s"
ONEOF_REPLACE="interface{} "

######################
# Remove the ManagementV1alpha1 prefix from the resources generated
######################
MV1_SEARCH="ManagementV1alpha1"
MV1_REPLACE=""

######################
# Remove the CatalogV1alpha1 prefix from the resources generated
######################
CV1_SEARCH="CatalogV1alpha1"
CV1_REPLACE=""

######################
# Replace and specifc Owner objects with the shared Owner structure
######################
OWNER_SEARCH="Owner.*\`json:\"owner,omitempty\"\`"
OWNER_REPLACE="Owner *apiv1.Owner \`json:\"owner,omitempty\"\`"
OWNER_IMPORT="import apiv1 \"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1\""
PACKAGE_SEARCH="^package.*$"

for file in ${MODELS}; do
    if grep -e ${TIME_SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${TIME_SEARCH}/i ${COMMENT}" ${file}
        # comment out the line we're changing
        $SED -i -e "s/${TIME_SEARCH}/\/\/ &/" ${file}
        # add in the new line we want
        $SED -i "/\/\/${TIME_SEARCH}/a ${TIME_REPLACE}" ${file}
    fi
    if grep -e ${ONEOF_SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${ONEOF_SEARCH}/i ${COMMENT}" ${file}
        # replace the Oneof type
        $SED -i -e "s/${ONEOF_SEARCH}/${ONEOF_REPLACE}/g" ${file}
    fi
    if grep -e ${MV1_SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${MV1_SEARCH}/i ${COMMENT}" ${file}
        # remove the prefix
        $SED -i -e "s/${MV1_SEARCH}/${MV1_REPLACE}/g" ${file}
    fi
    if grep -e ${CV1_SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${CV1_SEARCH}/i ${COMMENT}" ${file}
        # remove the prefix
        $SED -i -e "s/${CV1_SEARCH}/${CV1_REPLACE}/g" ${file}
    fi
    if grep -e ${OWNER_SEARCH} ${file} >> /dev/null; then
        # add a comment to the code
        $SED -i -e "/${OWNER_SEARCH}/i ${COMMENT}" ${file}
        # remove the prefix
        $SED -i -e "s/${OWNER_SEARCH}/${OWNER_REPLACE}/g" ${file}

        ## add a comment before and import the package
        $SED -i -e "/${PACKAGE_SEARCH}/a ${COMMENT}\n${OWNER_IMPORT}" ${file}
    fi
    # reformat the code
    go fmt ${file}
done
