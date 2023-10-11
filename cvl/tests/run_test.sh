#!/bin/bash

profiling=""
testcase=""
coverpkgs="-coverpkg=github.com/Azure/sonic-mgmt-common/cvl,github.com/Azure/sonic-mgmt-common/cvl/internal/util,github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"

if [ "${BUILD}:" != ":" ] ; then
	go test -mod=vendor -v -c -gcflags="all=-N -l" 
fi

if [ "${TESTCASE}:" != ":" ] ; then
	testcase="-run ${TESTCASE}"
fi

if [ "${PROFILE}:" != ":" ] ; then
	profiling="-bench=. -benchmem -cpuprofile profile.out"
fi

#Run test and display report
if [ "${NOREPORT}:" != ":" ] ; then
	go test -mod=vendor -v -tags test -cover ${coverpkgs} ${testcase}
elif [ "${COVERAGE}:" != ":" ] ; then
	go test -mod=vendor -v -tags test -cover -coverprofile coverage.out ${coverpkgs} ${testcase}
	go tool cover -html=coverage.out
else
	go test -mod=vendor -v -tags test -cover -json ${profiling} ${testcase} | tparse -smallscreen -all
fi

#With profiling 
#go test  -v -cover -json -bench=. -benchmem -cpuprofile profile.out | tparse -smallscreen -all

