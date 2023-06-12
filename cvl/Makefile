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
SRC_FILES=$(shell find . -name '*.go' | grep -v '_test.go' | grep -v '/tests/')
TEST_FILES=$(wildcard *_test.go)
TOP_DIR := ..
BUILD_DIR:=$(TOP_DIR)/build/cvl

CVL_SCHEMA_DIR = $(BUILD_DIR)/schema
CVL_SCHEMA     = $(CVL_SCHEMA_DIR)/.done
SONIC_YANG_DIR   = $(TOP_DIR)/build/yang/sonic
SONIC_YANG_FILES = $(shell find $(SONIC_YANG_DIR) -name '*.yang')
YANG_SRC_DIR     = ../models/yang

CVL_TEST_DIR = $(TOP_DIR)/build/tests/cvl
CVL_TEST_BIN = $(CVL_TEST_DIR)/cvl.test
CVL_TEST_SCHEMA_DIR = $(CVL_TEST_DIR)/testdata/schema
CVL_TEST_SCHEMA     = $(CVL_TEST_SCHEMA_DIR)/.done
CVL_TEST_YANGS      = $(wildcard testdata/schema/*.yang)
CVL_TEST_YANGS     += $(wildcard $(YANG_SRC_DIR)/sonic/common/*.yang)

DEFAULT_TARGETS = $(CVL_SCHEMA)
ifeq ($(NO_TEST_BINS),)
DEFAULT_TARGETS += $(CVL_TEST_BIN)
endif

all: $(DEFAULT_TARGETS)

.SECONDEXPANSION:

.PRECIOUS: %/.
%/.:
	mkdir -p $@

$(CVL_TEST_BIN): $(TEST_FILES) $(SRC_FILES) $(CVL_TEST_SCHEMA)
	cp -r testdata/*.json $(@D)/testdata
	$(GO) test -mod=vendor -cover -coverpkg=../cvl,../cvl/internal/util,../cvl/internal/yparser -c ../cvl -o $@

.PHONY: schema
schema: $(CVL_SCHEMA)

$(CVL_SCHEMA): $(SONIC_YANG_FILES) | $$(@D)/.
	$(TOP_DIR)/tools/pyang/generate_yin.py \
		--no-err-prefix \
		--path=$(SONIC_YANG_DIR) \
		--path=$(YANG_SRC_DIR)/common \
		--out-dir=$(@D)
	touch $@

.PHONY: test-schema
test-schema: $(CVL_TEST_SCHEMA)

$(CVL_TEST_SCHEMA): $(CVL_TEST_YANGS) | $$(@D)/.
	$(TOP_DIR)/tools/pyang/generate_yin.py \
		--no-err-prefix \
		--path=testdata/schema \
		--path=$(YANG_SRC_DIR)/common \
		--path=$(YANG_SRC_DIR)/sonic/common \
		--out-dir=$(@D)
	touch $@

gotest: $(CVL_TEST_SCHEMA)
	CVL_CFG_FILE=$(abspath .)/conf/cvl_cfg.json \
	CVL_SCHEMA_PATH=$(abspath $(CVL_TEST_SCHEMA_DIR)) \
	tests/run_test.sh

clean:
	$(RM) -r $(CVL_TEST_DIR) $(BUILD_DIR)

cleanall:clean

