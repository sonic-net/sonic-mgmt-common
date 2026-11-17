module example.com/management_common_consumer

go 1.25.9

require github.com/Azure/sonic-mgmt-common v0.0.0

replace github.com/golang/glog => github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b

// Use the management runtime versions from the source module.
replace golang.org/x/net => golang.org/x/net v0.53.0

replace golang.org/x/sys => golang.org/x/sys v0.43.0

replace golang.org/x/text => golang.org/x/text v0.39.0
