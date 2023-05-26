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

TOPDIR := ../..
BUILD_DIR := $(TOPDIR)/build

YANGDIR                     := $(BUILD_DIR)/yang
YANGDIR_COMMON              := $(YANGDIR)/common
YANGDIR_EXTENSIONS          := $(YANGDIR)/extensions
YANGDIR_ANNOTATIONS         := $(YANGDIR)/annotations

YANGDIR_SONIC               := $(YANGDIR)/sonic
YANGDIR_SONIC_COMMON        := $(YANGDIR_SONIC)/common

include sonic/import.mk
SONICYANG_IMPORT_DIR ?= $(or \
	$(wildcard $(TOPDIR)/../sonic-yang-models/cvlyang-models), \
	$(wildcard /usr/local/cvlyang-models) )

$(info SONICYANG_IMPORT_DIR = $(SONICYANG_IMPORT_DIR))

SONIC_YANG_IMPORT_FILES   := $(sort $(foreach pat,$(SONICYANG_IMPORTS),$(wildcard $(SONICYANG_IMPORT_DIR)/$(pat))))
SONIC_YANG_IMPORT_NAMES   := $(notdir $(SONIC_YANG_IMPORT_FILES))
SONIC_YANG_LOCAL_NAMES    := $(notdir $(wildcard sonic/*.yang))
SONIC_YANG_OVERLAPS       := $(filter $(SONIC_YANG_LOCAL_NAMES),$(SONIC_YANG_IMPORT_NAMES))
SHOULD_IMPORT_SONIC_YANGS := $(if $(SONIC_YANG_IMPORT_NAMES),y,)
SHOULD_IMPORT_SONIC_PORT  := $(if $(filter sonic-port.yang, $(SONIC_YANG_IMPORT_NAMES)),y,)

SYNC_STD_YANGS    := $(YANGDIR)/.sync_std_yangs
SYNC_EXT_YANGS    := $(YANGDIR)/.sync_ext_yangs
SYNC_SONIC_YANGS  := $(YANGDIR)/.sync_sonic_yangs
SYNC_ANNOT_YANGS  := $(YANGDIR)/.sync_annot_yangs
PREPARE_YANGS     := $(SYNC_STD_YANGS) $(SYNC_EXT_YANGS) $(SYNC_ANNOT_YANGS) $(SYNC_SONIC_YANGS)

YANG_ANNOTS_CHK   := $(YANGDIR)/.annots_checked

ifeq ($(INCLUDE_TEST_MODELS),y)
TEST_YANGS_DIR    ?= $(TOPDIR)/translib/transformer/test
TEST_API_YANGS    := $(filter-out %-annot.yang, $(wildcard $(TEST_YANGS_DIR)/openconfig-*.yang))
TEST_SONIC_YANGS  := $(filter-out %-annot.yang, $(wildcard $(TEST_YANGS_DIR)/sonic-*.yang))
TEST_ANNOT_YANGS  := $(wildcard $(TEST_YANGS_DIR)/*-annot.yang)
endif

TOOLS_DIR         := $(TOPDIR)/tools
PYANG_PLUGIN_DIR  := $(TOOLS_DIR)/pyang/pyang_plugins
PYANG             ?= pyang

ALL_TARGETS := allyangs.tree allyangs_tree.html
ALL_TARGETS += sonic_allyangs.tree sonic_allyangs_tree.html
ALL_TARGETS += #$(YANG_ANNOTS_CHK)

all: $(PREPARE_YANGS) $(ALL_TARGETS)


#======================================================================
# Syntax check for annotation files
#======================================================================

.PHONY: annot
annot: $(YANG_ANNOTS_CHK)

$(YANG_ANNOTS_CHK): $(PREPARE_YANGS)
	$(PYANG) --strict -f tree \
		-p $(YANGDIR_COMMON):$(YANGDIR):$(YANGDIR_EXTENSIONS):$(YANGDIR_SONIC) \
		$(YANGDIR_ANNOTATIONS)/*.yang > /dev/null
	touch $@

#======================================================================
# Yang tree for standard yangs and their extensions
#======================================================================

allyangs.tree: $(SYNC_STD_YANGS) $(SYNC_EXT_YANGS)
	$(PYANG) \
		-f tree \
		-o $@ \
		-p $(YANGDIR_COMMON):$(YANGDIR) \
		$(YANGDIR)/*.yang $$(find $(YANGDIR_EXTENSIONS) -name '*.yang')
	@echo "+++++ Generation of  YANG tree for Yang modules completed +++++"

allyangs_tree.html: $(SYNC_STD_YANGS) $(SYNC_EXT_YANGS)
	$(PYANG) \
		-f jstree \
		-o $@ \
		-p $(YANGDIR_COMMON):$(YANGDIR) \
		$(YANGDIR)/*.yang $$(find $(YANGDIR_EXTENSIONS) -name '*.yang')
	@echo "+++++ Generation of  HTML tree for Yang modules completed +++++"

#======================================================================
# Yang tree for SONiC yangs
#======================================================================

sonic_allyangs.tree: $(SYNC_SONIC_YANGS)
	$(PYANG) \
		-f tree \
		-o $@ \
		-p $(YANGDIR_SONIC_COMMON):$(YANGDIR_SONIC):$(YANGDIR_COMMON) \
		$(YANGDIR_SONIC)/*.yang
	@echo "+++++ Generation of  YANG tree for Sonic Yang modules completed +++++"

sonic_allyangs_tree.html: $(SYNC_SONIC_YANGS)
	$(PYANG) \
		-f jstree \
		-o $@ \
		-p $(YANGDIR_SONIC_COMMON):$(YANGDIR_SONIC):$(YANGDIR_COMMON) \
		$(YANGDIR_SONIC)/*.yang
	@echo "+++++ Generation of  HTML tree for Sonic Yang modules completed +++++"

#======================================================================
# Sync and patch yangs into build/yang directory
#======================================================================

.SECONDEXPANSION:

.PRECIOUS: %/. 
%/.:
	mkdir -p $@

$(SYNC_STD_YANGS): $(wildcard *.yang) $(wildcard common/*.yang) $(TEST_API_YANGS) | $$(@D)/.
	rsync -oavh --include='*/' --include='*.yang' --exclude='*' *.yang common $(TEST_API_YANGS) $(@D)
	touch $@

$(SYNC_EXT_YANGS): $(wildcard extensions/*.yang) | $$(@D)/.
	rsync -oavh --include='*/' --include='*.yang' --exclude='*' extensions $(@D)
	touch $@

$(SYNC_ANNOT_YANGS): $(wildcard annotations/*.yang) $(TEST_ANNOT_YANGS) | $$(@D)/.
	rsync -oavh annotations/*.yang $(TEST_ANNOT_YANGS) $(@D)/annotations
	touch $@

$(SYNC_SONIC_YANGS): $(wildcard sonic/*.yang) $(SONIC_YANG_IMPORT_FILES) $(TEST_SONIC_YANGS) | $$(@D)/sonic/common/.
ifneq ($(SHOULD_IMPORT_SONIC_YANGS),y)
	# Nothing to import, blindly copy all local sonic yangs
	rsync -oavh --include='*/' --include='*.yang' --exclude='*' sonic $(@D)
else
	# Copy everything from local sonic/common
	rsync -oavh sonic/common/*.yang $(@D)/sonic/common

	# Copy local sonic yangs excluding the ones specified in SONICYANG_IMPORTS
	$(TOPDIR)/tools/pyang/import_yang.py \
		--from=sonic --from=common \
		--to=$(@D)/sonic \
		$(SONIC_YANG_OVERLAPS:%=--exclude=%) \
		$(if $(SHOULD_IMPORT_SONIC_PORT),--subst-port-ref=ifname:name,) \
		--keep-comments \
		$(SONIC_YANG_LOCAL_NAMES)

	# Copy sonic yangs specified in SONICYANG_IMPORTS, from SONICYANG_IMPORT_DIR
	$(TOPDIR)/tools/pyang/import_yang.py \
		--from=$(SONICYANG_IMPORT_DIR) --from common \
		--to=$(@D)/sonic \
		$(if $(SHOULD_IMPORT_SONIC_PORT),,--subst-port-ref=name:ifname) \
		$(SONIC_YANG_IMPORT_NAMES)
endif
ifneq ($(TEST_SONIC_YANGS),)
	rsync -oavh $(TEST_SONIC_YANGS) $(@D)/sonic
endif
	touch $@

#======================================================================
# Cleanups
#======================================================================

clean:
	$(RM) $(ALL_TARGETS)
	$(RM) -r $(YANGDIR)

