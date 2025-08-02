// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package root

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"

	"github.com/luxfi/cli/v2/pkg/ansible"
	"github.com/luxfi/cli/v2/pkg/constants"
	"github.com/luxfi/cli/v2/pkg/models"
	"github.com/luxfi/cli/v2/pkg/ssh"
	"github.com/luxfi/cli/v2/tests/e2e/commands"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"golang.org/x/exp/slices"
)

const (
	luxdVersion = "v1.10.18"
	network            = "testnet"
	networkCapitalized = "Testnet"
	numNodes           = 2
	relativePath       = "nodes"
)

var (
	hostName         string
	NodeID           string
	monitoringHostID string
	createdHosts     []*models.Host
	// host names without docker prefix
	createdHostsFormatted []string
)

var _ = ginkgo.Describe("[Node monitoring]", func() {
	ginkgo.It("can create a node", func() {
		output := commands.NodeCreate(network, luxdVersion, numNodes, true, 0, commands.ExpectSuccess)
		fmt.Println(output)
		gomega.Expect(output).To(gomega.ContainSubstring("Luxd and Lux-CLI installed and node(s) are bootstrapping!"))
		// parse hostName
		re := regexp.MustCompile(`Generated staking keys for host (\S+)\[NodeID-(\S+)\]`)
		match := re.FindStringSubmatch(output)
		if len(match) >= 3 {
			hostName = match[1]
			NodeID = fmt.Sprintf("NodeID-%s", match[2])
		} else {
			ginkgo.Fail("failed to parse hostName and NodeID")
		}
	})
	ginkgo.It("creates cluster config", func() {
		usr, err := user.Current()
		gomega.Expect(err).Should(gomega.BeNil())
		homeDir := usr.HomeDir
		content, err := os.ReadFile(filepath.Join(homeDir, constants.E2EBaseDirName, relativePath, constants.ClustersConfigFileName))
		gomega.Expect(err).Should(gomega.BeNil())
		clustersConfig := models.ClustersConfig{}
		err = json.Unmarshal(content, &clustersConfig)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(clustersConfig.Clusters[constants.E2EClusterName].Network.Kind.String()).To(gomega.Equal(networkCapitalized))
		gomega.Expect(clustersConfig.Clusters[constants.E2EClusterName].Nodes).To(gomega.HaveLen(numNodes))
		monitoringHostID = clustersConfig.Clusters[constants.E2EClusterName].MonitoringInstance
		createdHostsFormatted = append(createdHostsFormatted, clustersConfig.Clusters[constants.E2EClusterName].Nodes...)
	})
	ginkgo.It("checks prometheus config in monitoring host", func() {
		usr, err := user.Current()
		gomega.Expect(err).Should(gomega.BeNil())
		homeDir := usr.HomeDir
		monitoringHost, err := ansible.GetInventoryFromAnsibleInventoryFile(filepath.Join(homeDir, constants.E2EBaseDirName, relativePath, constants.AnsibleInventoryDir, "e2e", "monitoring"))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(monitoringHost).To(gomega.HaveLen(1))
		err = ssh.RunSSHDownloadNodePrometheusConfig(monitoringHost[0], filepath.Join(homeDir, constants.E2EBaseDirName, relativePath, monitoringHostID))
		gomega.Expect(err).Should(gomega.BeNil())
		createdDockerHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(filepath.Join(homeDir, constants.E2EBaseDirName, relativePath, constants.AnsibleInventoryDir, "e2e"))
		gomega.Expect(err).Should(gomega.BeNil())
		createdHosts = createdDockerHosts
		hostluxdPorts := []string{}
		hostMachinePorts := []string{}
		for _, host := range createdHosts {
			hostluxdPorts = append(hostluxdPorts, fmt.Sprintf("%s:9650", host.IP))
			hostMachinePorts = append(hostMachinePorts, fmt.Sprintf("%s:9100", host.IP))
		}
		prometheusConfig := commands.ParsePrometheusYamlConfig(filepath.Join(homeDir, constants.E2EBaseDirName, relativePath, monitoringHostID, constants.NodePrometheusConfigFileName))
		scrapeConfig := prometheusConfig.ScrapeConfigs
		luxdJob := "luxd"
		luxdMachineJob := "luxd-machine"
		for _, newConfig := range scrapeConfig {
			if newConfig.JobName == luxdJob || newConfig.JobName == luxdMachineJob {
				targets := newConfig.StaticConfigs
				dockerTarget := targets[0]
				gomega.Expect(len(dockerTarget.Targets)).To(gomega.Equal(numNodes))
				if newConfig.JobName == luxdJob {
					for _, host := range hostluxdPorts {
						gomega.Expect(slices.Contains(dockerTarget.Targets, host)).To(gomega.Equal(true))
					}
				} else {
					for _, host := range hostMachinePorts {
						gomega.Expect(slices.Contains(dockerTarget.Targets, host)).To(gomega.Equal(true))
					}
				}
			}
		}
	})
	ginkgo.It("verifies prometheus metrics configured on cluster hosts", func() {
		for _, host := range createdHosts {
			sshOutput := commands.NodeSSH(host.IP, "sudo systemctl status prometheus")
			gomega.Expect(sshOutput).To(gomega.ContainSubstring("Active: active (running)"))
		}
	})
	ginkgo.It("verifies node exporter metrics configured on cluster hosts", func() {
		for _, host := range createdHosts {
			sshOutput := commands.NodeSSH(host.IP, "sudo systemctl status node_exporter")
			gomega.Expect(sshOutput).To(gomega.ContainSubstring("Active: active (running)"))
		}
	})
	ginkgo.It("verifies promtail metrics configured on monitoring host", func() {
		sshOutput := commands.NodeSSH(monitoringHostID, "sudo systemctl status promtail")
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("Active: active (running)"))
	})
	ginkgo.It("verifies loki metrics configured on monitoring host", func() {
		sshOutput := commands.NodeSSH(monitoringHostID, "sudo systemctl status loki")
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("Active: active (running)"))
	})
	ginkgo.It("verifies correct promtail config", func() {
		usr, err := user.Current()
		gomega.Expect(err).Should(gomega.BeNil())
		homeDir := usr.HomeDir
		monitoringHost, err := ansible.GetInventoryFromAnsibleInventoryFile(filepath.Join(homeDir, constants.E2EBaseDirName, relativePath, constants.AnsibleInventoryDir, "e2e", "monitoring"))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(monitoringHost).To(gomega.HaveLen(1))
		sshOutput := commands.NodeSSH(monitoringHostID, "sudo cat /etc/promtail/promtail.yml")
		gomega.Expect(sshOutput).To(gomega.ContainSubstring(fmt.Sprintf("url: http://%s:%d/loki/api/v1/push", monitoringHost[0].IP, constants.LuxdLokiPort)))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("tenant_id: lux"))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("CF-Access-Client-Id: lux"))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("__path__: /home/ubuntu/.luxd/logs/C.log"))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("__path__: /home/ubuntu/.luxd/logs/P.log"))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("__path__: /home/ubuntu/.luxd/logs/X.log"))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("__path__: /home/ubuntu/.luxd/logs/main.log"))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("__path__: /home/ubuntu/loadtest_*.txt"))
	})
	ginkgo.It("verifies correct loki config", func() {
		usr, err := user.Current()
		gomega.Expect(err).Should(gomega.BeNil())
		homeDir := usr.HomeDir
		monitoringHost, err := ansible.GetInventoryFromAnsibleInventoryFile(filepath.Join(homeDir, constants.E2EBaseDirName, relativePath, constants.AnsibleInventoryDir, "e2e", "monitoring"))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(monitoringHost).To(gomega.HaveLen(1))
		sshOutput := commands.NodeSSH(monitoringHostID, "sudo cat /etc/loki/loki.yml")
		gomega.Expect(sshOutput).To(gomega.ContainSubstring(fmt.Sprintf("http_listen_port: %d", constants.LuxdLokiPort)))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("chunks_directory: /var/lib/loki/chunks"))
		gomega.Expect(sshOutput).To(gomega.ContainSubstring("store: tsdb"))
	})
	ginkgo.It("configured luxd", func() {
		luxdConfig := commands.NodeSSH(constants.E2EClusterName, "cat /home/ubuntu/.luxd/configs/node.json")
		gomega.Expect(luxdConfig).To(gomega.ContainSubstring("\"network-id\": \"" + network + "\""))
		gomega.Expect(luxdConfig).To(gomega.ContainSubstring("public-ip"))
		luxdConfigCChain := commands.NodeSSH(constants.E2EClusterName, "cat /home/ubuntu/.luxd/configs/chains/C/config.json")
		gomega.Expect(luxdConfigCChain).To(gomega.ContainSubstring("\"state-sync-enabled\": true"))
	})
	ginkgo.It("provides luxd with staking certs", func() {
		stakingFiles := commands.NodeSSH(constants.E2EClusterName, "ls /home/ubuntu/.luxd/staking/")
		gomega.Expect(stakingFiles).To(gomega.ContainSubstring("signer.key"))
		gomega.Expect(stakingFiles).To(gomega.ContainSubstring("staker.crt"))
		gomega.Expect(stakingFiles).To(gomega.ContainSubstring("staker.key"))
	})
	ginkgo.It("can get cluster status", func() {
		output := commands.NodeStatus()
		fmt.Println(output)
		gomega.Expect(output).To(gomega.ContainSubstring("Checking if node(s) are bootstrapped to Primary Network"))
		gomega.Expect(output).To(gomega.ContainSubstring("Checking if node(s) are healthy"))
		gomega.Expect(output).To(gomega.ContainSubstring("Getting luxd version of node(s)"))
		gomega.Expect(output).To(gomega.ContainSubstring(constants.E2ENetworkPrefix))
		gomega.Expect(output).To(gomega.ContainSubstring(hostName))
		gomega.Expect(output).To(gomega.ContainSubstring(NodeID))
		gomega.Expect(output).To(gomega.ContainSubstring(networkCapitalized))
	})
	ginkgo.It("can ssh to a created node", func() {
		output := commands.NodeSSH(constants.E2EClusterName, "echo hello")
		gomega.Expect(output).To(gomega.ContainSubstring("hello"))
	})
	ginkgo.It("can list created nodes", func() {
		output := commands.NodeList()
		fmt.Println(output)
		gomega.Expect(output).To(gomega.ContainSubstring(networkCapitalized))
		gomega.Expect(output).To(gomega.ContainSubstring("docker1"))
		gomega.Expect(output).To(gomega.ContainSubstring("NodeID"))
		gomega.Expect(output).To(gomega.ContainSubstring(constants.E2ENetworkPrefix))
	})
	ginkgo.It("logged operations", func() {
		logs := commands.NodeSSH(constants.E2EClusterName, "cat /home/ubuntu/.luxd/logs/main.log")
		gomega.Expect(logs).To(gomega.ContainSubstring("initializing node"))
		gomega.Expect(logs).To(gomega.ContainSubstring("initializing API server"))
		gomega.Expect(logs).To(gomega.ContainSubstring("creating leveldb"))
		gomega.Expect(logs).To(gomega.ContainSubstring("initializing database"))
		gomega.Expect(logs).To(gomega.ContainSubstring("creating proposervm wrapper"))
		gomega.Expect(logs).To(gomega.ContainSubstring("check started passing"))
	})
	ginkgo.It("can cleanup", func() {
		commands.DeleteE2EInventory()
		commands.DeleteE2ECluster()
		for _, host := range createdHostsFormatted {
			commands.DeleteNode(host)
		}
		commands.DeleteNode(monitoringHostID)
	})
})
