#######################################################################
#
# Copyright 2019 Broadcom. All rights reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
#
#######################################################################

TOPDIR    ?= ..
BUILD_DIR ?= $(TOPDIR)/build

GO ?= go
export GO

TRANSLIB_BLD = $(BUILD_DIR)/translib
FORMAT_CHECK = $(TRANSLIB_BLD)/.formatcheck

SRCS  := $(shell find . -name '*.go' -not -name '*_test.go' -not -path '*/test/*' -not -path '*/ocbinds/*')
SRCX  := $(shell find ocbinds -name '*.go' -not -name 'ocbinds*.go')
TESTS := $(shell find . -name '*_test.go' -not -path '*/test/*')

TRANSLIB_MAIN_SRCS = $(filter-out ./transformer/%, $(SRCS) $(SRCX))
TRANSLIB_TEST_SRCS = $(filter-out ./transformer/% ./db/%, $(TESTS))
TRANSL_DB_ALL_SRCS = $(filter ./db/%, $(SRCS) $(TESTS))

TRANSLIB_TEST_DIR  = $(BUILD_DIR)/tests/translib
TRANSLIB_TEST_BIN  = $(TRANSLIB_TEST_DIR)/translib.test
TRANSL_DB_TEST_BIN = $(TRANSLIB_TEST_DIR)/db.test

TRANSFORMER_TEST_BIN = $(TRANSLIB_TEST_DIR)/transformer.test
TRANSFORMER_ALL_SRCS = $(filter ./transformer/%, $(SRCS) $(TESTS))

TRANSFORMER_TEST_APP_BIN = $(TRANSLIB_TEST_DIR)/testapp.test
YANG_DIR   = ../build/yang
YANG_FILES = $(shell find $(YANG_DIR) -name "*.yang" -not -path "*/annotations/*")
YGOT_BINDS = ocbinds/ocbinds.go

SONIC_DEP_YANGS  = $(notdir $(wildcard $(YANG_DIR)/sonic/common/*.yang))
YGOT_EXCLUDES    = ietf-interfaces sonic-types sonic-common

XFMR_BUILD_DIR   = $(BUILD_DIR)/transformer
XFMR_MODELS_LIST = $(XFMR_BUILD_DIR)/models_list
XFMR_TEST_MODELS = $(notdir $(wildcard transformer/test/*.yang))

DEFAULT_TARGETS = $(YGOT_BINDS) $(XFMR_MODELS_LIST) $(FORMAT_CHECK)
ifeq ($(NO_TEST_BINS),)
DEFAULT_TARGETS += $(TRANSLIB_TEST_BIN) $(TRANSL_DB_TEST_BIN) $(TRANSFORMER_TEST_APP_BIN)
ifdef INCLUDE_TEST_MODELS
DEFAULT_TARGETS += $(TRANSFORMER_TEST_BIN)
endif
endif

all: $(DEFAULT_TARGETS)

.PRECIOUS: %/.
%/.:
	mkdir -p $@

$(TRANSLIB_TEST_BIN): $(TRANSLIB_MAIN_SRCS) $(TRANSLIB_TEST_SRCS) $(YGOT_BINDS)
	$(GO) test -mod=vendor -tags test -cover -coverpkg=../translib,../translib/tlerr -c ../translib -o $@

$(TRANSL_DB_TEST_BIN) : $(TRANSL_DB_ALL_SRCS)
	$(GO) test -mod=vendor -cover -c ../translib/db -o $@

$(TRANSFORMER_TEST_BIN): $(TRANSFORMER_ALL_SRCS) $(TRANSLIB_MAIN_SRCS) $(YGOT_BINDS)
	$(GO) test -mod=vendor -tags xfmrtest -c -vet=off -cover -coverpkg=../translib/transformer ../translib/transformer -o $@

$(TRANSFORMER_TEST_APP_BIN): $(TRANSFORMER_ALL_SRCS) $(TRANSLIB_MAIN_SRCS) $(YGOT_BINDS)
	$(GO) test -mod=vendor -tags testapp -c -vet=off -cover -coverpkg=../translib/transformer ../translib/transformer -o $@

$(YGOT_BINDS): $(YANG_FILES)
	$(RM) $@
	$(GO) run \
		--mod=vendor \
		$(TOPDIR)/vendor/github.com/openconfig/ygot/generator/generator.go \
		--logtostderr \
		--output_file=$@ \
		--package_name=ocbinds \
		--generate_fakeroot \
		--fakeroot_name=device \
		--compress_paths=false \
		--path=$(YANG_DIR) \
		--exclude_modules=$(shell printf "%s," $(basename $(YGOT_EXCLUDES))) \
		$^

$(XFMR_MODELS_LIST): $(TOPDIR)/config/transformer/models_list $(YANG_FILES)
	mkdir -p $(@D)
	$(RM) $@ $(@D)/extra_models
	printf -- "-%s\n" $(SONIC_DEP_YANGS) > $(@D)/extra_models
ifeq ($(INCLUDE_TEST_MODELS),y)
	printf -- "%s\n" $(XFMR_TEST_MODELS) >> $(@D)/extra_models
endif
	cat $< $(@D)/extra_models > $@

.SECONDEXPANSION:

$(FORMAT_CHECK): $(SRCS) $(SRCX) $(TESTS) | $(YGOT_BINDS) $$(@D)/.
	$(TOPDIR)/tools/test/format-check.sh \
		--log=$(TRANSLIB_BLD)/formatcheck.log \
		$?
	touch $@

clean:
	$(RM) $(YGOT_BINDS)
	$(RM) -r $(TRANSLIB_TEST_DIR) $(XFMR_BUILD_DIR)
