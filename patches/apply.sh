#!/usr/bin/env bash

set -e

PATCH_DIR=$(dirname $(realpath ${BASH_SOURCE[0]}))

DEST_DIR=vendor
[ ! -z $1 ] && DEST_DIR=$1

if [ ! -d "${DEST_DIR}" ]; then
    echo "Unknown DEST_DIR \"${DEST_DIR}\""
    exit 1
fi

# Copy some of the packages from go mod download directory into vendor directory.
# It is a workaround for 'go mod vendor' not copying all files

[ -z ${GO} ] && GO=go
[ -z ${GOPATH} ] && GOPATH=$(${GO} env GOPATH)
PKGPATH=$(echo ${GOPATH} | sed 's/:.*$//g')/pkg/mod

# Copy package files from GOPATH/pkg/mod to vendor
# $1 = package name, $2 = version, $3... = files
function copy() {
    for FILE in "${@:3}"; do
        rsync -r --chmod=u+w --exclude=testdata --exclude=*_test.go \
            ${PKGPATH}/$1@$2/${FILE}  ${DEST_DIR}/$1/
    done
}

set -x

copy github.com/openconfig/ygot v0.7.1 ygen genutil generator

copy github.com/openconfig/goyang v0.0.0-20200309174518-a00bece872fc .

copy github.com/openconfig/gnmi v0.0.0-20200617225440-d2b4e6a45802 .

# Update vendor/modules.txt to include the copied packages
# This is needed for Go 1.24+ which validates modules.txt even for --mod=vendor
function update_modules_txt() {
    local modules_file="${DEST_DIR}/modules.txt"

    # Add gnmi module entries if not already present
    if ! grep -q '^github.com/openconfig/gnmi/ctree$' "${modules_file}" 2>/dev/null; then
        echo "" >> "${modules_file}"
        echo "# github.com/openconfig/gnmi v0.0.0-20200617225440-d2b4e6a45802" >> "${modules_file}"
        echo "## explicit" >> "${modules_file}"
        echo "github.com/openconfig/gnmi/ctree" >> "${modules_file}"
        echo "github.com/openconfig/gnmi/errlist" >> "${modules_file}"
        echo "github.com/openconfig/gnmi/proto/gnmi" >> "${modules_file}"
    fi

    # Add ygot module entries if not already present
    if ! grep -q '^github.com/openconfig/ygot/ygen$' "${modules_file}" 2>/dev/null; then
        echo "" >> "${modules_file}"
        echo "# github.com/openconfig/ygot v0.7.1" >> "${modules_file}"
        echo "## explicit" >> "${modules_file}"
        echo "github.com/openconfig/ygot/genutil" >> "${modules_file}"
        echo "github.com/openconfig/ygot/generator" >> "${modules_file}"
        echo "github.com/openconfig/ygot/ygen" >> "${modules_file}"
    fi
}

update_modules_txt

# Apply patches

patch -d ${DEST_DIR}/github.com/openconfig/ygot -p1 < ${PATCH_DIR}/ygot/ygot.patch

patch -d ${DEST_DIR}/github.com/openconfig/goyang -p1 < ${PATCH_DIR}/goyang/goyang.patch

patch -d ${DEST_DIR}/github.com/antchfx/jsonquery -p1 < ${PATCH_DIR}/jsonquery.patch
patch -d ${DEST_DIR}/github.com/antchfx/xmlquery -p1 < ${PATCH_DIR}/xmlquery.patch
patch -d ${DEST_DIR}/github.com/antchfx/xpath -p1 < ${PATCH_DIR}/xpath.patch

patch -d ${DEST_DIR}/github.com/golang/glog  -p1 < ${PATCH_DIR}/glog.patch

