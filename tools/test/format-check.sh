#!/usr/bin/env bash
################################################################################
#                                                                              #
#  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
#  its subsidiaries.                                                           #
#                                                                              #
#  Licensed under the Apache License, Version 2.0 (the "License");             #
#  you may not use this file except in compliance with the License.            #
#  You may obtain a copy of the License at                                     #
#                                                                              #
#     http://www.apache.org/licenses/LICENSE-2.0                               #
#                                                                              #
#  Unless required by applicable law or agreed to in writing, software         #
#  distributed under the License is distributed on an "AS IS" BASIS,           #
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.    #
#  See the License for the specific language governing permissions and         #
#  limitations under the License.                                              #
#                                                                              #
################################################################################

set -e
function print_help_and_exit() {
    echo "usage: format-check.sh [OPTIONS] [SRC_PATH...]"
    echo ""
    echo "OPTIONS:"
    echo " -exclude=DIR  Directory to exclude for format checks. It can be repeated"
    echo " -log=FILE     Write format checker logs to a file (and to stdout)."
    echo " -fix          Fixes the formatting in the files."
    echo ""
    echo "SRC_PATH selects source files or directories for format analysis."
    echo "If SRC_PATH is not specified, whole current directory tree is included."
    exit 0
}

# Format checker options
EXCLUDE=( vendor build patches ocbinds_*.go )
FORMAT=false

SRC_PATH=()

while [[ $# -gt 0 ]]; do
case "$1" in
    -exclude=*|--exclude=*)
        EXCLUDE+=( "$(echo $1 | cut -d= -f2-)" )
        shift ;;
    -log=*|--log=*)
        LOGFILE="$(echo $1 | cut -d= -f2-)"
        shift ;;
    -fix|--fix)
        FORMAT=true
        shift ;;
    -v) VERBOSE=y; shift;;
    -*) print_help_and_exit ;;
    *)  [[ -d $1 ]] && SRC_PATH+=( "$1/*.go" ) || SRC_PATH+=( "$1" ); shift;;
esac
done

if [[ -z ${SRC_PATH} ]]; then
    EX=()
    for E in ${EXCLUDE[@]}; do
        [[ $E == *.go ]] || E="$E/*"
        EX+=( -not -path "*/$E" )
    done
    SRC_PATH+=( $(find . -name '*.go' "${EX[@]}" ) )
fi

[[ -z $GO ]] && export GO=go
[[ -z $GOPATH ]] && export GOPATH=/tmp/go
export GOBIN=$(echo ${GOPATH} | sed 's/:.*$//g')/bin
export PATH=$($GO env GOROOT)/bin:${PATH}
GOFMT=${GOFMT:-gofmt}

# Create a temporary logfile if not specified thru -log option.
[[ -z ${LOGFILE} ]] && LOGFILE=$(mktemp -t gofmt.XXXXX)

[[ -z ${VERBOSE} ]] || echo -e "Source files: ${SRC_PATH[@]}\n"

if [ "$FORMAT" = true ] ; then
	${GOFMT} -w ${SRC_PATH[@]} | tee ${LOGFILE}
else
	${GOFMT} -l ${SRC_PATH[@]} | tee ${LOGFILE}
fi

NUM_ERROR=$(< "$LOGFILE" wc -l)
[[ ${NUM_ERROR} == 0  ]] || cat << EOM

${NUM_ERROR} files have formatting errors; listed in ${LOGFILE}
Execute '$(basename $0) -fix <FILES>' to fix issues.
EOM

test $((NUM_ERROR)) -lt 1
