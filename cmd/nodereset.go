package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/platform9/nodeadm/pkg/logrus"

	"github.com/platform9/nodeadm/constants"
	"github.com/platform9/nodeadm/systemd"
	"github.com/platform9/nodeadm/utils"
	"github.com/spf13/cobra"
)

// nodeCmd represents the cluster command
var nodeCmdReset = &cobra.Command{
	Use:   "reset",
	Short: "Reset node to clean up all kubernetes install and configuration",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Fail on first error instead of best effort cleanup
		cleanupKeepalived()
		kubeadmReset()
		cleanupKubelet()
		cleanupBinaries()
		cleanupNetworking()
		cleanupDockerImages()
	},
}

func kubeadmReset() {
	log.Infof("[nodeadm:reset] Invoking kubeadm reset")
	_ = exec.Command(filepath.Join(constants.BaseInstallDir, "kubeadm"), "reset", "--ignore-preflight-errors=all").Run()
}

func cleanupKeepalived() {
	log.Infof("[nodeadm:reset] Stopping & Removing Keepalived")
	if err := systemd.StopIfActive("keepalived.service"); err != nil {
		log.Fatalf("Failed to stop keepalived service: %v", err)
	}
	if err := systemd.DisableIfEnabled("keepalived.service"); err != nil {
		log.Fatalf("Failed to disable keepalived service: %v", err)
	}
	os.RemoveAll(filepath.Join(constants.SystemdDir, "keepalived.service"))
	os.Remove(constants.KeepalivedConfigFilename)
}

func cleanupKubelet() {
	log.Infof("[nodeadm:reset] Stopping & Removing kubelet")
	if err := systemd.StopIfActive("kubelet.service"); err != nil {
		log.Fatalf("Failed to stop kubelet service: %v", err)
	}
	if err := systemd.DisableIfEnabled("kubelet.service"); err != nil {
		log.Fatalf("Failed to disable kubelet service: %v", err)
	}
	failed, err := systemd.Failed("kubelet.service")
	if err != nil {
		log.Fatalf("Failed to check if kubelet service failed: %v", err)
	}
	if failed {
		if err := systemd.ResetFailed("kubelet.service"); err != nil {
			log.Fatalf("Failed to reset failed kubelet service: %v", err)
		}
	}
	os.RemoveAll(filepath.Join(constants.SystemdDir, "kubelet.service"))
	os.RemoveAll(filepath.Join(constants.SystemdDir, "kubelet.service.d"))
}

func cleanupBinaries() {
	log.Infof("[nodeadm:reset] Removing kubernetes binaries")
	os.RemoveAll(filepath.Join(constants.BaseInstallDir, "kubelet"))
	os.RemoveAll(filepath.Join(constants.BaseInstallDir, "kubeadm"))
	os.RemoveAll(filepath.Join(constants.BaseInstallDir, "kubectl"))

	os.RemoveAll(constants.CNIBaseDir)
}

func cleanupNetworking() {
	log.Infof("[nodeadm:reset] Removing flannel state files & resetting networking")
	os.RemoveAll(constants.CNIConfigDir)
	os.RemoveAll(constants.CNIStateDir)
	_ = exec.Command("ip", "link", "del", "cni0").Run()
	_ = exec.Command("ip", "link", "del", "flannel.1").Run()
}

func cleanupDockerImages() {
	for _, image := range utils.GetImages() {
		_ = exec.Command("docker", "rmi", image).Run()
	}
}

func init() {
	rootCmd.AddCommand(nodeCmdReset)
}
