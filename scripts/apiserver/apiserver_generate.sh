#!/bin/bash

export OUTDIR=/codegen/output

# clean existing models and clients files
cwd=`pwd`
cd ${OUTDIR}/models/
rm -r `ls | grep -v "api"`
cd ${OUTDIR}/clients/
rm -r `ls | grep -v "api"`
cd ${cwd}

# default to prod
if [ -z "${PROTOCOL}" ]; then export PROTOCOL=https; fi
if [ -z "${HOST}" ]; then export HOST=apicentral.axway.com; fi
if [ -z "${PORT}" ]; then export PORT=443; fi

# set the environment vars
export GO_POST_PROCESS_FILE="`command -v gofmt` -w"
export GO111MODULE=on

openapi-generator-cli version-manager set 4.3.1

if node ./generate.js ${PROTOCOL} ${HOST} ${PORT}; then
  # update all go imports
  goimports -w=true ${OUTDIR}

  # run script to modify any files that need tweaking
  ./modify_models.sh

  # copy over the fake example test file
  cp ./fake_example_test.tmpl ${OUTDIR}/clients/management/v1alpha1/fake_example_test.go

  # replace the access control model files
  cp ./model_access_control_list_spec.tmpl ${OUTDIR}/models/catalog/v1alpha1/model_access_control_list_spec.go
  cp ./model_access_control_list_spec.tmpl ${OUTDIR}/models/definitions/v1alpha1/model_access_control_list_spec.go
  cp ./model_access_control_list_spec.tmpl ${OUTDIR}/models/management/v1alpha1/model_access_control_list_spec.go

  chown -R ${USERID}:${GROUPID} ${OUTDIR}
else
  echo "FAILED: generating resources"
fi

rm ./openapitools.json