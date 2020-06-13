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

YANGDIR                     := .
YANGDIR_COMMON              := $(YANGDIR)/common
YANGDIR_EXTENSIONS          := $(YANGDIR)/extensions
YANGDIR_ANNOTATIONS         := $(YANGDIR)/annotations
YANG_MOD_FILES              := $(wildcard *.yang)
YANG_MOD_FILES              += $(wildcard $(YANGDIR_EXTENSIONS)/*.yang)
YANG_COMMON_FILES           := $(wildcard $(YANGDIR_COMMON)/*.yang)
YANG_ANNOTATION_FILES       := $(wildcard $(YANGDIR_ANNOTATIONS)/*.yang)

YANGDIR_SONIC               := $(YANGDIR)/sonic
YANGDIR_SONIC_COMMON        := $(YANGDIR_SONIC)/common
SONIC_YANG_MOD_FILES        := $(wildcard $(YANGDIR_SONIC)/*.yang)
SONIC_YANG_COMMON_FILES     := $(wildcard $(YANGDIR_SONIC_COMMON)/*.yang)

YANG_ANNOTS_CHK   := .annots_checked

TOOLS_DIR         := $(TOPDIR)/tools
PYANG_PLUGIN_DIR  := $(TOOLS_DIR)/pyang/pyang_plugins
PYANG             ?= pyang

ALL_TARGETS := allyangs.tree allyangs_tree.html
ALL_TARGETS += sonic_allyangs.tree sonic_allyangs_tree.html
ALL_TARGETS += #$(YANG_ANNOTS_CHK)

all: $(ALL_TARGETS)


#======================================================================
# Syntax check for annotation files
#======================================================================

.PHONY: annot
annot: $(YANG_ANNOTS_CHK)

$(YANG_ANNOTS_CHK): $(YANG_ANNOTATION_FILES) $(YANG_MOD_FILES) $(YANG_COMMON_FILES)
	$(PYANG) --strict -f tree \
		-p $(YANGDIR_COMMON):$(YANGDIR):$(YANGDIR_EXTENSIONS):$(YANGDIR_SONIC) \
		$(YANG_ANNOTATION_FILES) > /dev/null
	touch $@

#======================================================================
# Yang tree for standard yangs and their extensions
#======================================================================

allyangs.tree:  $(YANG_MOD_FILES) $(YANG_COMMON_FILES)
	$(PYANG) \
		-f tree \
		-o $@ \
		-p $(YANGDIR_COMMON):$(YANGDIR) \
		$(YANG_MOD_FILES)
	@echo "+++++ Generation of  YANG tree for Yang modules completed +++++"

allyangs_tree.html:  $(YANG_MOD_FILES) $(YANG_COMMON_FILES)
	$(PYANG) \
		-f jstree \
		-o $@ \
		-p $(YANGDIR_COMMON):$(YANGDIR) \
		$(YANG_MOD_FILES)
	@echo "+++++ Generation of  HTML tree for Yang modules completed +++++"

#======================================================================
# Yang tree for SONiC yangs
#======================================================================

sonic_allyangs.tree: $(SONIC_YANG_MOD_FILES) $(SONIC_YANG_COMMON_FILES)
	$(PYANG) \
		-f tree \
		-o $@ \
		-p $(YANGDIR_SONIC_COMMON):$(YANGDIR_SONIC):$(YANGDIR_COMMON) \
		$(SONIC_YANG_MOD_FILES)
	@echo "+++++ Generation of  YANG tree for Sonic Yang modules completed +++++"

sonic_allyangs_tree.html: $(SONIC_YANG_MOD_FILES) $(SONIC_YANG_COMMON_FILES)
	$(PYANG) \
		-f jstree \
		-o $@ \
		-p $(YANGDIR_SONIC_COMMON):$(YANGDIR_SONIC):$(YANGDIR_COMMON) \
		$(SONIC_YANG_MOD_FILES)
	@echo "+++++ Generation of  HTML tree for Sonic Yang modules completed +++++"

#======================================================================
# Cleanups
#======================================================================

clean:
	$(RM) $(ALL_TARGETS)

