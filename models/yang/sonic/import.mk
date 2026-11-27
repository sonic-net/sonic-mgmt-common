
##
# SONICYANG_IMPORTS is the list of sonic yang files to be copied
# from SONICYANG_IMPORT_DIR. Only the file basenames (like sonic-sflow.yang)
# or glob patterns of basenames (like sonic-telemetry*.yang) can be specified.
# Other sonic yangs referred by these will also be copied.
#
SONICYANG_IMPORTS += sonic-sflow.yang
SONICYANG_IMPORTS += sonic-interface.yang
SONICYANG_IMPORTS += sonic-port.yang
SONICYANG_IMPORTS += sonic-portchannel.yang
SONICYANG_IMPORTS += sonic-vlan.yang
SONICYANG_IMPORTS += sonic-mclag.yang
SONICYANG_IMPORTS += sonic-types.yang
SONICYANG_IMPORTS += sonic-vrf.yang

# sonic-syslog is the ConfigDB SYSLOG_SERVER / SYSLOG_CONFIG schema (TLS
# protocol, tls_profile_id, VRF, filters, FEATURE rate-limits). Import it so
# CVL validates translib writes driven by OpenConfig logging the same way
# sonic-yang-mgmt validates config CLI. TLS apply-state leaves stay in the
# local sonic-syslog-state augment (see SONIC_YANG_DEFERRED_NAMES).
SONICYANG_IMPORTS += sonic-syslog.yang

##
# Local sonic yangs that import modules from SONICYANG_IMPORTS must be copied
# after those imports are synced into build/yang/sonic.
#
# Augments imported sonic-syslog with STATE_DB TLS apply-state leaves.
SONIC_YANG_DEFERRED_NAMES += sonic-syslog-state.yang