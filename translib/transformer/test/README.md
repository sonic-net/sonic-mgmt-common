# Transformer Unit Testing

Following are the instructions on how to build and execute transformer unit test.
The test folder (sonic-mgmt-common/translib/transformer/test) contains all the necessary files to build and execute transformer test binary that includes test-yangs, annotations, callbacks to serve the annotations, unit-test file to exercise the annotations and transformer unit test frameowrk files.

* Generate transformer.test by building sonic-mgmt-common
* Copy the openconfig-test-xfmr.yang, openconfig-test-xfmr-annot.yang, sonic-test-xfmr.yang, sonic-test-xfmr-annot.yang to  mgmt-framework docker /usr/models/yang directory
* Edit the models_list file in mgmt-framework docker /usr/models/yang directory to include openconfig-test-xfmr.yang, openconfig-test-xfmr-annot.yang and sonic-test-xfmr-annot.yang files
* Copy the sonic-test-xfmr.yin file from sonic-mgmtcommon/build/cvl/schema/ to /usr/sbin/schema/ in mgmt-framework docker 
* Copy the transformer.test binary to mgmt-frameowrk docker in /usr/sbin directory and then execute : 
```shell
(./transformer.test -test.v -test.coverprofile=transformer.cover -logtostderr -v=5 | tee transformer.out ) >& transformer.log
```
* View the results in file transformer.out (All test-cases should have PASS prefix)
* View the transformer.log file to view debug logs for debugging the test-case failures if any.