
SUBDIRS=

# SDI-API New API headers
HEADERS+=inc/sdi_entity_info.h inc/sdi_media.h
HEADERS+=inc/sdi_fan.h inc/sdi_led.h inc/sdi_thermal.h inc/sdi_entity.h

include ${MAKE_INC}/workspace.mak
