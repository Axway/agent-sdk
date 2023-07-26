#!/bin/bash

sonar-scanner -X \
    -Dsonar.host.url=${SONAR_HOST_URL} \
    -Dsonar.language=go \
    -Dsonar.projectName=APIC_AGENTS_SDK \
    -Dsonar.projectVersion=1.0 \
    -Dsonar.projectKey=APIC_AGENTS_SDK \
    -Dsonar.sourceEncoding=UTF-8 \
    -Dsonar.projectBaseDir=${WORKSPACE} \
    -Dsonar.sources=. \
    -Dsonar.tests=. \
	-Dsonar.exclusions=**/mock/**,**/testdata/**,**/apiserver/clients/**,**/apiserver/models/**,**/api/v1/**,**/mock*.go,**/*.json,**/definitions.go,**/errors.go,**/error.go,**/proto/**,**/samples/**,**/*.pb.go \
    -Dsonar.test.inclusions=**/*test*.go \
    -Dsonar.go.tests.reportPaths=goreport.json \
    -Dsonar.go.coverage.reportPaths=gocoverage.out \
    -Dsonar.issuesReport.console.enable=true \
    -Dsonar.report.export.path=sonar-report.json

exit 0
