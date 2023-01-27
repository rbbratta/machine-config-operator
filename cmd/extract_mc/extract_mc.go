package main

import (
	"github.com/openshift/machine-config-operator/pkg/daemon"
	"os"
)

func main() {

	err := daemon.ExtractMachineConfig(os.Args[1])
	if err != nil {
		panic(err)
	}
}
