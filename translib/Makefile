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

TRANSLIB_PKG = $(BUILD_DIR)/pkg/linux_amd64/translib.a

TRANSLIB_MAIN_SRCS = $(shell find . -name '*.go' | grep -v '_test.go' | grep -v '/test/')
TRANSLIB_TEST_SRCS = $(shell find . -maxdepth 1 -name '*_test.go')
TRANSL_DB_ALL_SRCS = $(shell find db/ -name '*.go' | grep -v '/test/')

TRANSLIB_TEST_DIR  = $(BUILD_DIR)/tests/translib
TRANSLIB_TEST_BIN  = $(TRANSLIB_TEST_DIR)/translib.test
TRANSL_DB_TEST_BIN = $(TRANSLIB_TEST_DIR)/db.test

YANG_FILES = $(shell find $(TOPDIR)/models/yang -name '*.yang')
YGOT_BINDS = ocbinds/ocbinds.go

DEFAULT_TARGETS = $(YGOT_BINDS)
ifeq ($(NO_TEST_BINS),)
DEFAULT_TARGETS += $(TRANSLIB_TEST_BIN) $(TRANSL_DB_TEST_BIN)
endif

default: $(DEFAULT_TARGETS)

all: $(DEFAULT_TARGETS) $(TRANSLIB_PKG)

$(TRANSLIB_PKG): $(TRANSLIB_MAIN_SRCS) $(YGOT_BINDS)
	$(GO) build -mod=vendor -gcflags="all=-N -l" -v -o $@ ../translib

$(TRANSLIB_TEST_BIN): $(TRANSLIB_MAIN_SRCS) $(TRANSLIB_TEST_SRCS) $(YGOT_BINDS)
	$(GO) test -mod=vendor -cover -coverpkg=../translib,../translib/tlerr -c ../translib -o $@

$(TRANSL_DB_TEST_BIN) : $(TRANSL_DB_ALL_SRCS)
	$(GO) test -mod=vendor -cover -c ../translib/db -o $@

$(YGOT_BINDS): $(YANG_FILES)
	$(RM) $@
	cd ocbinds && $(GO) generate

clean:
	$(RM) $(YGOT_BINDS)
	$(RM) $(TRANSLIB_PKG)
	$(RM) -r $(TRANSLIB_TEST_DIR)

