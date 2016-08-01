SONiC SDI API
-------------
This repo contains all API declarations for System Device Interface API for the SONiC PAS component.


Description
-----------

This repo has all public API declarations for the SDI API.  The PAS component utilizes the SDI API to get/set data for platform devices be it fans, power supplies, leds, qsfps, etc..

The implementations of the API can be very platform specific and are kept in a seperate repository(ies) to facilitate reuse if these with diff arch implementation.
examples directory has Reference for implementing any new SDI functionality.  

Building
--------
Please see the instructions in the sonic-nas-manifest repo for more details on the common build tools.  [Sonic-nas-manifest](https://github.com/Azure/sonic-nas-manifest)

Development Dependencies:
 -sonic-logging
 -sonic-common-utils

Dependent Packages:
 - libsonic-logging1 libsonic-logging-dev libsonic-common1 libsonic-common-dev


BUILD CMD: sonic_build --dpkg libsonic-logging1 libsonic-logging-dev libsonic-common1 libsonic-common-dev  -- clean binary

(c) Dell 2016
