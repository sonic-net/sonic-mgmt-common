#!/usr/bin/env bash

function print_usage() {
cat << EOM
usage: $(basename $0) [OPTIONS] [TESTARGS]

OPTIONS:
  -pkg PACKAGE    Test package name. Should be translib or its child package.
                  Defaults to translib.
  -run PATTERN    Testcase pattern. Equivalent of 'go test -run PATTERN ...'
  -bench PATTERN  Benchmark pattern. Only one of -run or -bench is allowed.
                  Equivalent of 'go test -bench PATTERN -benchmem -run ^$ ...'
  -json           Dump test logs in json format. Output can be piped to tools
                  like tparse or gotestsum.
  -vet=off        Equivalent to -vet=off option.
  -tags BLDTAGS   Comma separated build tags to use. Defaults to "test"

TESTARGS:         Any other arguments to be passed to TestMain. All values that
                  do not match above listed options are treated as test args.
                  Equivalent of 'go test ... -args TESTARGS'

EOM
}

set -e

TOPDIR=$(git rev-parse --show-toplevel)
GO=${GO:-go}

TARGS=( -mod=vendor -cover )
PARGS=()
PKG=translib
TAG=test

while [[ $# -gt 0 ]]; do
    case "$1" in
    -h|-help|--help)  print_usage; exit 0;;
    -p|-pkg|-package) PKG=$2; shift 2;;
    -r|-run)   TARGS+=( -run "$2" ); shift 2;;
    -b|-bench) TARGS+=( -bench "$2" -benchmem -run "^$" ); shift 2;;
    -j|-json)  TARGS+=( -json ); ECHO=0; shift;;
    -vet=off)  TARGS+=( -vet=off ); shift;;
    -tags)     TAG="$2"; shift 2;;
    *) PARGS+=( "$1"); shift;;
    esac
done

cd ${TOPDIR}
if [[ ! -d ${PKG} ]] && [[ -d translib/${PKG} ]]; then
    PKG=translib/${PKG}
fi

if [[ -z ${GOPATH} ]]; then
    export GOPATH=/tmp/go
fi

export $(${TOPDIR}/tools/test/env.sh --dest=${TOPDIR}/build/test | xargs)

[[ ${TARGS[*]} =~ -bench ]] || TARGS+=( -v )
[[ -z ${TAG} ]] || TARGS+=( -tags ${TAG} )
[[ "${PARGS[@]}" =~ -(also)?log* ]] || PARGS+=( -logtostderr )

[[ ${ECHO} == 0 ]] || set -x
${GO} test ./${PKG} "${TARGS[@]}" -args "${PARGS[@]}"
