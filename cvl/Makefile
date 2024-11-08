################################################################################
#                                                                              #
#  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
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

GO?=go
TOPDIR ?= ..
BUILD_DIR:=$(TOPDIR)/build/cvl
FORMAT_CHECK = $(BUILD_DIR)/.formatcheck
SRC_FILES=$(shell find . -name '*.go' | grep -v '_test.go' | grep -v '/tests/')
TEST_FILES=$(wildcard *_test.go)
CVL_TEST_DIR = $(TOPDIR)/build/tests/cvl
CVL_TEST_BIN = $(CVL_TEST_DIR)/cvl.test
CVL_TEST_SCHEMA_DIR = $(CVL_TEST_DIR)/testdata/schema
CVL_TEST_CONFIG = $(CVL_TEST_DIR)/cvl_cfg.json
CVL_TEST_DB_CONFIG = $(CVL_TEST_DIR)/database_config.json

CVL_SCHEMA_DIR = $(BUILD_DIR)/schema
CVL_SCHEMA = $(CVL_SCHEMA_DIR)/.done
SONIC_YANG_DIR   = $(TOPDIR)/build/yang/sonic
SONIC_YANG_FILES = $(shell find $(SONIC_YANG_DIR) -name '*.yang')
YANG_SRC_DIR     = ../models/yang

SONIC_YANG_COMMON := $(TOPDIR)/models/yang/sonic/common
CVL_TEST_SCHEMA   := $(CVL_TEST_SCHEMA_DIR)/.done
CVL_TEST_YANGS     = $(shell find testdata/schema -name '*.yang')
CVL_TEST_YANGS    += $(wildcard $(SONIC_YANG_COMMON)/*.yang)

DEFAULT_TARGETS = $(CVL_SCHEMA) $(FORMAT_CHECK)
ifdef DEBUG
	GOFLAGS += -gcflags="all=-N -l"
endif

ifeq ($(NO_TEST_BINS),)
DEFAULT_TARGETS += $(CVL_TEST_BIN) $(CVL_TEST_SCHEMA)
endif

all: $(DEFAULT_TARGETS)

.PRECIOUS: %/.
%/.:
	mkdir -p $@

.SECONDEXPANSION:

.PHONY: schema
schema: $(CVL_SCHEMA)

$(CVL_SCHEMA): $(SONIC_YANG_FILES) | $$(@D)/.
	tools/generate_yin.py \
		--path=$(SONIC_YANG_DIR) \
		--path=$(YANG_SRC_DIR)/common \
		--out-dir=$(@D)
	touch $@

$(CVL_TEST_BIN): $(TEST_FILES) $(SRC_FILES) | $$(@D)/testdata/.
	cp -r testdata/*.json $(@D)/testdata
	$(GO) test -mod=vendor -tags=test -cover -coverpkg=../cvl,../cvl/internal/util,../cvl/internal/yparser -c ../cvl -o $@

.PHONY: test-schema
test-schema: $(CVL_TEST_SCHEMA)

$(CVL_TEST_SCHEMA): $(CVL_TEST_YANGS) | $$(@D)/.
	tools/generate_yin.py \
		--path=testdata/schema \
		--path=$(YANG_SRC_DIR)/common \
		--path=$(YANG_SRC_DIR)/sonic/common \
		--out-dir=$(@D)
	touch $@

$(CVL_TEST_CONFIG): conf/cvl_cfg.json
	sed -E 's/((TRACE|LOG).*)\"false\"/\1\"true\"/' conf/cvl_cfg.json > $@

$(CVL_TEST_DB_CONFIG): $(TOPDIR)/tools/test/database_config.json
	$(TOPDIR)/tools/test/dbconfig.py -o $@

gotest: $(CVL_TEST_SCHEMA) $(CVL_TEST_CONFIG) $(CVL_TEST_DB_CONFIG)
	CVL_CFG_FILE=$(abspath $(CVL_TEST_CONFIG)) \
	CVL_SCHEMA_PATH=$(CVL_TEST_SCHEMA_DIR) \
	DB_CONFIG_PATH=$(abspath $(CVL_TEST_DB_CONFIG)) \
	tests/run_test.sh

$(FORMAT_CHECK): $(SRC_FILES) $(TEST_FILES) | $$(@D)/.
	$(TOPDIR)/tools/test/format-check.sh \
		--log=$(@D)/formatcheck.log \
		$?
	touch $@

clean:
	make -C tests clean
	$(RM) -r $(BUILD_DIR)
	$(RM) -r $(wildcard $(PKG_BUILD_DIR)/*/cvl)
	$(RM) -r $(CVL_TEST_DIR)

cleanall:clean

