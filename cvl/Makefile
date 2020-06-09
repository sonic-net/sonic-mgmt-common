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
CVL_PKG=$(TOP_DIR)/build/pkg/linux_amd64/cvl.a

CVL_TEST_DIR = $(TOP_DIR)/build/tests/cvl
CVL_TEST_BIN = $(CVL_TEST_DIR)/cvl.test
CVL_TEST_SCHEMA_DIR = $(CVL_TEST_DIR)/testdata/schema

DEFAULT_TARGETS = deps schema
ifeq ($(NO_TEST_BINS),)
DEFAULT_TARGETS += $(CVL_TEST_BIN) test-schema
endif

default: $(DEFAULT_TARGETS)

all: $(DEFAULT_TARGETS) $(CVL_PKG)

deps: 
	mkdir -p $(BUILD_DIR)

$(CVL_PKG): $(SRC_FILES)
	@echo "Building $@"
	$(GO) build -mod=vendor -o $(CVL_PKG) ../cvl

$(CVL_TEST_BIN): $(TEST_FILES) $(SRC_FILES) | test-schema
	cp -r testdata/*.json $(@D)/testdata
	$(GO) test -mod=vendor -cover -coverpkg=../cvl,../cvl/internal/util,../cvl/internal/yparser -c ../cvl -o $@

.PHONY: schema
schema:
	$(MAKE) -C schema

test-schema:
	$(MAKE) -C testdata/schema

tests:
	$(MAKE) -C tests

gotest:schema test-schema
	CVL_CFG_FILE=$(abspath .)/conf/cvl_cfg.json CVL_SCHEMA_PATH=$(CVL_TEST_SCHEMA_DIR) tests/run_test.sh

clean:
	$(MAKE) -C tests clean
	$(MAKE) -C schema clean
	$(MAKE) -C testdata/schema clean
	$(RM) -r $(CVL_PKG)
	$(RM) -r $(CVL_TEST_DIR)

