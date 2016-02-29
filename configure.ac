#                                               -*- Autoconf -*-
# Process this file with autoconf to produce a configure script.

AC_PREREQ([2.69])
AC_INIT([sonic-logging], [1.0], [clifford_wichmann@dell.com])
AM_INIT_AUTOMAKE([subdir-objects])
AC_CONFIG_SRCDIR([config.h.in])
AC_CONFIG_HEADERS([config.h])
AC_CHECK_HEADERS([libxml/parser.h])

# Checks for programs.
AC_PROG_CXX
AC_PROG_CC
LT_INIT([shared])
# Checks for libraries.

# Checks for header files.
AC_CHECK_HEADERS([fcntl.h stdlib.h string.h syslog.h unistd.h libxml2/libxml/parser.h])

# Checks for typedefs, structures, and compiler characteristics.
AC_CHECK_HEADER_STDBOOL
AC_C_INLINE
AC_TYPE_SIZE_T

# Checks for library functions.
AC_FUNC_MMAP
AC_FUNC_STRERROR_R
AC_CHECK_FUNCS([ftruncate mkdir strncasecmp])

AC_CONFIG_FILES([Makefile])
AC_OUTPUT
