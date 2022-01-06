#!/bin/bash
MODE=

if [ "$1" = "preview" ]; then
    MODE=preview
fi

sonar-scanner -X \
    -Dsonar.host.url=${SONAR_HOST} \
    -Dsonar.language=go \
    -Dsonar.projectName=APIC_AGENTS_SDK \
    -Dsonar.projectVersion=1.0 \
    -Dsonar.projectKey=APIC_AGENTS_SDK \
    -Dsonar.sourceEncoding=UTF-8 \
    -Dsonar.projectBaseDir=${WORKSPACE} \
    -Dsonar.sources=. \
    -Dsonar.tests=. \
	-Dsonar.exclusions=**/mock/**,**/testdata/**,**/apiserver/clients/**,**/apiserver/models/**,**/api/v1/**,**/mock*.go,**/*.json,**/definitions.go,**/errors.go,**/error.go,**/proto/**,**/samples/** \
    -Dsonar.test.inclusions=**/*test*.go \
    -Dsonar.go.tests.reportPaths=goreport.json \
    -Dsonar.go.coverage.reportPaths=gocoverage.out \
    -Dsonar.analysis.mode=${MODE} \
    -Dsonar.issuesReport.console.enable=true \
    -Dsonar.report.export.path=sonar-report.json

if [ "${MODE}" = "preview" ]; then
    echo "All reported issues:" && cat .scannerwork/sonar-report.json | jq ".issues"

    ISSUES=$(cat .scannerwork/sonar-report.json | jq '.issues | .[] | select(.severity=="BLOCKER")' | jq -s length)
    echo "BLOCKING Issues: ${ISSUES}"

    ISSUES=$(cat .scannerwork/sonar-report.json | jq '.issues | .[] | select(.severity=="CRITICAL")' | jq -s length)
    echo "CRITICAL Issues: ${ISSUES}"

    ISSUES=$(cat .scannerwork/sonar-report.json | jq '.issues | .[] | select(.severity=="MAJOR")' | jq -s length)
    echo "MAJOR Issues: ${ISSUES}"

    ISSUES=$(cat .scannerwork/sonar-report.json | jq '.issues | .[] | select(.severity=="MINOR")' | jq -s length)
    echo "MINOR Issues: ${ISSUES}"

    HIGH_SEV_ISSUES=$(cat .scannerwork/sonar-report.json | jq '.issues | .[] | (select(.severity=="MAJOR"),select(.severity=="BLOCKER"),select(.severity=="CRITICAL"))' | jq -s length)
    echo "All issues with a severity of MAJOR or higher: ${HIGH_SEV_ISSUES}"

    if [ ${HIGH_SEV_ISSUES} -gt 0 ]; then
        exit 1
    fi
fi

exit 0
