BUILT_SOURCES := $(top_srcdir)/src/event_log_levels.c
CLEANFILES = $(top_srcdir)/src/event_log_levels.c

$(top_srcdir)/src/event_log_levels.c : $(top_srcdir)/inc/event_log_types.h
	python $(top_srcdir)/src/enum_automation.py $(top_srcdir)/inc/event_log_types.h $(top_srcdir)/inc/priv_event_log_utils.h $@

lib_LTLIBRARIES = libsonic_logging.la

libsonic_logging_la_SOURCES = src/event_log_cli.c src/event_log_utils.c \
     src/event_log.c         src/event_log_types.c

nodist_libsonic_logging_la_SOURCES= src/event_log_levels.c

libsonic_logging_la_LDFLAGS = 
libsonic_logging_la_CFLAGS = -I$(top_srcdir)/inc -I$(includedir)/libxml2


