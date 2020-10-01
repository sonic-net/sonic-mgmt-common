# YANG directory

## Directory structure

    yang/               --> Standard YANGs
    |-- annotations/    --> Transformer annotations
    |-- common/         --> Dependencies for standard YANGs
    |-- extensions/     --> Extenstions for standard YANGs
    |-- sonic/          --> SONiC yangs
    |-- testdata/       --> Test YANGs - ignored
    `-- version.xml     --> YANG bundle version configuration file

All supported standard YANG files (OpenConfig and IETF) are kept in this **yang** directory. Usual practice is to keep only top level YANG module here and keep dependent YANGs, submodules in **yang/common** directory.

Example: openconfig-platform.yang is kept in top **yang** directory and openconfig-platform-types.yang in **yang/common** directory.

All extenstion YANGs **MUST** be kept in **yang/extensions** directory.

## version.xml

version.xml file maintains the yang bundle version number in **Major.Minor.Patch** format.
It is the collective version number for all the YANG modules defined here.
**UPDATE THIS VERSION NUMBER FOR EVERY YANG CHANGE.**

**Major version** should be incremented if YANG model is changed in a non backward compatible manner.
Such changes should be avoided.

* Delete, rename or relocate data node
* Change list key attributes
* Change data type of a node to an incompatible type
* Change leafref target

**Minor version** should be incremented if the YANG change modifies the API in a backward
compatible way. Patch version should be reset to 0.
Candidate YANG changes for this category are:

* Add new YANG module
* Add new YANG data nodes
* Mark a YANG data node as deprecated
* Change data type of a node to a compatible type
* Add new enum or identity

**Patch version** should incremented for cosmetic fixes that do not change YANG API.
Candidate YANG changes for this category are:

* Change description, beautification.
* Expand pattern or range of a node to wider set.
* Change must expression to accept more cases.
* Error message or error tag changes.


