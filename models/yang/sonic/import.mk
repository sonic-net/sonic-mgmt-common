
##
# SONICYANG_IMPORTS is the list of sonic yang files to be copied
# from SONICYANG_IMPORT_DIR. Only the file basenames (like sonic-sflow.yang)
# or glob patterns of basenames (like sonic-telemetry*.yang) can be specified.
# Other sonic yangs referred by these will also be copied.
#
SONICYANG_IMPORTS += sonic-sflow.yang
SONICYANG_IMPORTS += sonic-interface.yang
SONICYANG_IMPORTS += sonic-port.yang