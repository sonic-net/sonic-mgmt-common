#!/usr/bin/env bash
################################################################################
#                                                                              #
#  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
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

[[ -z ${TOPDIR} ]] && TOPDIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../..)
[[ -z ${MAKE}   ]] && MAKE=make

YANGDIR=${TOPDIR}/models/yang

if [ -z $1 ]; then
    echo "usage: $0 YANG_FILE_NAME..."
    exit -1
fi

# Download, patch and compile goyang
${MAKE} -s -C ${TOPDIR} annotgen

# Run goyang to generate annotation file for the specified yang file.
# Annotation output is dumped on stdout.
${TOPDIR}/build/bin/goyang --format=annotate --path=${YANGDIR} "$@"
