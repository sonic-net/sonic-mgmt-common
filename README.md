## How to execute and clean up unit-tests in sonic-mgmt-common:
* `make -f makefile.test`: to execute all unit tests
* `make -f makefile.test clean`: clean up all the generated build artifacts
* `make -f makefile.test container`: build the test container image. The file `container` contains the container image information

## How to build unit-test container with customer
Note, it is best to `make -f makefile.test clean` before running a test with a customized library.

### With a new LIBYANG version in ../../target/debs/bullseye/
* `LIBYANG=1.0.74 make -f makefile.test`
### With a LIBYANG 1.0.75 in the user home directory
* `LIBYANG=1.0.74 SONIC_TARGET_DEBS=~ make -f makefile.test`
### With a sonic_yang_models in the user home directory
* `SONIC_TARGET_WHL=~ make -f makefile.test`

