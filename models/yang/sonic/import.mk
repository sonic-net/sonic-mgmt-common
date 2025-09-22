
##
# SONICYANG_IMPORTS is the list of sonic yang files to be copied
# from SONICYANG_IMPORT_DIR. Only the file basenames (like sonic-sflow.yang)
# or glob patterns of basenames (like sonic-telemetry*.yang) can be specified.
# Other sonic yangs referred by these will also be copied.
#

ifneq ($(SONIC_YANG_IMPORTS),)
SONICYANG_IMPORTS = $(shell echo $(SONIC_YANG_IMPORTS))
endif

SONICYANG_IMPORTS += sonic-sflow.yang
SONICYANG_IMPORTS += sonic-interface.yang
SONICYANG_IMPORTS += sonic-port.yang
SONICYANG_IMPORTS += sonic-portchannel.yang
SONICYANG_IMPORTS += sonic-vlan.yang
SONICYANG_IMPORTS += sonic-mclag.yang
SONICYANG_IMPORTS += sonic-types.yang
SONICYANG_IMPORTS += sonic-*vrf*.yang
SONICYANG_IMPORTS += sonic*system*.yang
SONICYANG_IMPORTS += sonic-device_metadata.yang
SONICYANG_IMPORTS += sonic-banner.yang
SONICYANG_IMPORTS += sonic-versions.yang
SONICYANG_IMPORTS += sonic-ssh-server.yang
SONICYANG_IMPORTS += sonic-syslog.yang
SONICYANG_IMPORTS += sonic-ntp.yang
SONICYANG_IMPORTS += sonic-dns.yang
SONICYANG_IMPORTS += sonic-system-aaa.yang
SONICYANG_IMPORTS += sonic-system-tacacs.yang
SONICYANG_IMPORTS += sonic-system-radius.yang
