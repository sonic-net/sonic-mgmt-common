package main

import (
	"fmt"
	"github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
)

func main() {
	fmt.Printf("owner libraries: %d %T %T\n", cvl.CONFIG_DB, translib.GetRequest{}, ocbinds.Device{})
}
