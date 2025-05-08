#!/bin/bash -x

# Run sanity tests for sonic-mgmt-common.
# Assumes sonic-mgmt-common is already compiled and all dependencies
# are installed.

STATUS=0
DEBDIR=$(realpath debian/sonic-mgmt-common)

OUTPUT_DIR=./
DETAILED_COV=n
SKIP_BUILD=n
while getopts 'dhto:' opt; do
    case "$opt" in
        t)
            echo "Running test only"
            SKIP_BUILD=y
            ;;
        o)
            echo "Coverage HTML directory"
            OUTPUT_DIR="$OPTARG"
            ;;
        d)
            DETAILED_COV=y
            ;;
        ?|h)
            echo "Usage: $(basename $0) [-t] [-d] [-o dir]"
            exit 100
            ;;
    esac
done

# build debian packages
INCLUDE_TEST_MODELS=y dpkg-buildpackage -rfakeroot -us -uc -b -j$(nproc)
if [ "$?" -ne "0" ];then
    echo "Error!!! Compilation failed"
    exit 1
fi

function generate_html_report() {
    /usr/local/go/bin/go tool cover -html=$1 -o ${OUTPUT_DIR}/$1.html
}

redis_ready=$(ps aux | grep -ie "redis-server" | grep -ie bin)
if [ -z "$redis_ready" ]; then
    sudo service redis-server start
    echo "Starting redis-server status: $?"
else
    echo "redis-server already started: $redis_ready"
fi

# Update unixsocket path in database_config.json
tools/test/dbconfig.py -o ${PWD}/build/tests/database_config.json
export DB_CONFIG_PATH=${PWD}/build/tests/database_config.json

# Run CVL tests
pushd build/tests/cvl
CVL_SCHEMA_PATH=testdata/schema \
    ./cvl.test -test.v -alsologtostderr -test.coverprofile coverage.cvl || STATUS=1
generate_html_report coverage.cvl
popd

# Run translib tests
pushd build/tests/translib
export CVL_SCHEMA_PATH=${DEBDIR}/usr/sbin/schema
export YANG_MODELS_PATH=${DEBDIR}/usr/models/yang
./db.test -test.v -alsologtostderr -test.coverprofile coverage.db || STATUS=2
generate_html_report coverage.db

# Populates test data in essential tables like PORT, DEVICE_METADATA, SWITCH_TABLE, USER_TABLE etc.
${PWD}/../../../tools/test/dbinit.py --overwrite

./translib.test -test.v -alsologtostderr -test.coverprofile coverage.translib || STATUS=3
generate_html_report coverage.translib

./testapp.test -test.v -alsologtostderr -test.coverprofile coverage.transformer  || STATUS=4
./transformer.test -test.v -alsologtostderr -test.coverprofile coverage.transformer  || STATUS=5
generate_html_report coverage.transformer
popd
exit ${STATUS}
