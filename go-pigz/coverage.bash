#!/usr/bin/env bash

set -e

COVERAGE_FILE=build/cover.out
COVERAGE_ANALYSIS_FILE=build/cover.analysis
COVERAGE_ANALYSIS_FILE_XML=build/coverage.xml

mkdir -p build
go test -covermode=atomic -coverprofile ${COVERAGE_FILE}
go tool cover -func=${COVERAGE_FILE} -o ${COVERAGE_ANALYSIS_FILE}
gocover-cobertura < ${COVERAGE_FILE} > ${COVERAGE_ANALYSIS_FILE_XML}
go tool cover -html=${COVERAGE_FILE}
