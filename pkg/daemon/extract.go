package daemon

import (
	"fmt"
	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ctrlcommon "github.com/openshift/machine-config-operator/pkg/controller/common"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog/v2"
	"os"
)

func newExtraceMockDaemon() Daemon {
	// Create a Daemon instance with mocked clients
	return Daemon{
		mock:             true,
		name:             "nodeName",
		kubeClient:       k8sfake.NewSimpleClientset(),
		bootedOSImageURL: "test",
	}
}

func extractWritiFiles(files []ign3types.File) error {
	cwd, _ := os.Getwd()
	fmt.Printf("%s\n", cwd)
	for _, file := range files {

		// We don't support appends in the file section, so instead of waiting to fail validation,
		// let's explicitly fail here.
		if len(file.Append) > 0 {
			return fmt.Errorf("found an append section when writing files. Append is not supported")
		}

		decodedContents, err := ctrlcommon.DecodeIgnitionFileContents(file.Contents.Source, file.Contents.Compression)
		if err != nil {
			return fmt.Errorf("could not decode file %q: %w", file.Path, err)
		}

		mode := defaultFilePermissions
		if file.Mode != nil {
			mode = os.FileMode(*file.Mode)
		}

		// trim leading '/' to extract locally
		if file.Path[0] == '/' {
			file.Path = file.Path[1:]
		}
		fmt.Printf("%s\t%s\n", mode, file.Path)
		//os.Stdout.Write(decodedContents)
		// set chown if file information is provided
		//uid, gid, err := getFileOwnership(file)
		//if err != nil {
		//	return fmt.Errorf("failed to retrieve file ownership for file %q: %w", file.Path, err)
		//}
		uid := os.Geteuid()
		gid := os.Getegid()
		if err := writeFileAtomically(file.Path, decodedContents, defaultDirectoryPermissions, mode, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

func ExtractMachineConfig(onceFrom string) error {

	dn := newExtraceMockDaemon()
	dn.skipReboot = false
	configi, _, err := dn.senseAndLoadOnceFrom(onceFrom)
	if err != nil {
		klog.Warningf("Unable to decipher onceFrom config type: %s", err)
		return err
	}
	switch c := configi.(type) {
	//case ign3types.Config:
	//	glog.V(2).Info("Daemon running directly from Ignition")
	//	return dn.runOnceFromIgnition(c)
	case mcfgv1.MachineConfig:
		// mutate MachineConfig
		klog.V(2).Info("Daemon running directly from MachineConfig")
		fmt.Printf("%v\n", c.Name)
		newIgnConfig, err := ctrlcommon.ParseAndConvertConfig(c.Spec.Config.Raw)
		if err != nil {
			return err
		}
		if err := extractWritiFiles(newIgnConfig.Storage.Files); err != nil {
			return err
		}
		return nil

	}

	return fmt.Errorf("unsupported onceFrom type provided")
}
