module example.com/management_common_consumer

go 1.26.0

require github.com/Azure/sonic-mgmt-common v0.0.0

// Use the management runtime versions from the source module.
replace golang.org/x/net => golang.org/x/net v0.62.0

replace golang.org/x/sys => golang.org/x/sys v0.49.0

replace golang.org/x/text => golang.org/x/text v0.44.0
