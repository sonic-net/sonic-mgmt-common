#!/usr/bin/env bash

function usage() {
cat << EOM
usage: $(basename $0) [OPTIONS]

Options:
  --dest=DIR            Copy yang and othermgmt_cmn files here. Creates a temporary directorty
                        if not specified.
  --dbconfig-in=FILE    Copy the database_config.json file from this file.
                        Unix socket paths will be updated by reading the redis configs.
                        Uses tools/test/database_config.json if not specified.
  --dbconfig=FILE       Use this database_config.json instead of creating new one under DEST.

EOM
}

set -e

mgmt_cmn=$(git -C $(dirname ${BASH_SOURCE}) rev-parse --show-toplevel)

while [[ $# != 0 ]]; do
    case "$1" in
    -h|--help)       usage; exit 0;;
    --dest=*)        DEST=${1#*=} ;;
    --dbconfig-in=*) DBCONFIG_IN="${1#*=}" ;;
    --dbconfig=*)    export DB_CONFIG_PATH="${1#*=}" ;;
    *) >&2 echo "error: unknown option \"$1\""; exit 1;;
    esac
    shift
done

if [[ -z ${DEST} ]]; then
    DEST=$(mktemp -d /tmp/translib.XXXXX)
elif [[ ! -d ${DEST} ]]; then
    mkdir -p ${DEST}
fi

# Create database_config.json if not specified through --dbconfig
if [[ -z ${DB_CONFIG_PATH} ]]; then
    export DB_CONFIG_PATH=${DEST}/database_config.json
fi
if [[ ! -e ${DB_CONFIG_PATH} ]] || [[ -n ${DBCONFIG_IN} ]]; then
    ${mgmt_cmn}/tools/test/dbconfig.py \
        -s ${DBCONFIG_IN:-${mgmt_cmn}/tools/test/database_config.json} \
        -o ${DB_CONFIG_PATH}
fi


# Prepare yang files directiry for transformer
if [[ -z ${YANG_MODELS_PATH} ]]; then
    export YANG_MODELS_PATH=${DEST}/yangs
fi
mkdir -p $V ${YANG_MODELS_PATH}
pushd ${YANG_MODELS_PATH} > /dev/null
rm -rf *
find ${mgmt_cmn}/build/yang -name "*.yang" -exec ln -sf {} \;
ln -sf ${mgmt_cmn}/models/yang/version.xml
ln -sf ${mgmt_cmn}/build/transformer/models_list
popd > /dev/null


# Setup CVL schema directory
if [[ -z ${CVL_SCHEMA_PATH} ]]; then
    export CVL_SCHEMA_PATH=${mgmt_cmn}/build/cvl/schema
fi

# Prepare CVL config file with all traces enabled
if [[ -z ${CVL_CFG_FILE} ]]; then
    export CVL_CFG_FILE=${DEST}/cvl_cfg.json
    if [[ ! -e ${CVL_CFG_FILE} ]]; then
        F=${mgmt_cmn}/cvl/conf/cvl_cfg.json
        sed -E 's/((TRACE|LOG).*)\"false\"/\1\"true\"/' $F > ${CVL_CFG_FILE}
    fi
fi


cat << EOM
DB_CONFIG_PATH=${DB_CONFIG_PATH}
YANG_MODELS_PATH=${YANG_MODELS_PATH}
CVL_SCHEMA_PATH=${CVL_SCHEMA_PATH}
CVL_CFG_FILE=${CVL_CFG_FILE}

EOM