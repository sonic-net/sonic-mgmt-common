# Networking-dell-sdi-api
Dell networking source code for network switch hardware
This repo has Dell SDI external API's which are used to configure devices via SDI layer.

Also has dell-platform_1.0.0.deb which has Dell S6000-On platform specific binaries. Until ACS Jenkins build project builds the repos.

ACS Target Testing.
1) Run OS10 base only on ACS  generic image.
2) Install libxml2
3) copy dell-platform-xx.deb built on docker image to / as root user and  do dpkg -i dell-platform-xx.deb
4) /opt/ngos/bin/csp_api_service &
5) /opt/ngos/bin/dn_pas_svc
6)  open a ssh session for ACS target and then use cps get/set operations (ex:- show-env,os10_media_show)
