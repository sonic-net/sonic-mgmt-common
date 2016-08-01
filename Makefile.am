
#All exported headers
nobase_include_HEADERS=sonic/sdi_entity.h sonic/sdi_entity_info.h \
            sonic/sdi_fan.h sonic/sdi_led.h sonic/sdi_media.h \
            sonic/sdi_thermal.h

# Build sdi stub library example to link against PAS
lib_LTLIBRARIES = libsonic_sdi_stub.la
libsonic_sdi_stub_la_SOURCES = examples/sdi_startup.c examples/sdi_entity.c examples/sdi_entity_framework.c examples/sdi_entity_info.c \
                               examples/sdi_entity_reset.c examples/sdi_fan.c examples/sdi_thermal.c \
                               examples/sdi_led.c examples/sdi_media.c

libsonic_sdi_stub_la_LDFLAGS = -lsonic_common -lsonic_logging -lpthread -lrt
libsonic_sdi_stub_la_CFLAGS =  -Isonic -I$(includedir)/sonic




