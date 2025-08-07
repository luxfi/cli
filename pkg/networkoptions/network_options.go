// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkoptions

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/pflag"

	"github.com/luxfi/cli/cmd/flags"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	sdkutils "github.com/luxfi/sdk/utils"
	// "github.com/luxfi/node/api/info" // TODO: Uncomment when custom endpoint support is added

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

type NetworkOption int64

const (
	Undefined NetworkOption = iota
	Mainnet
	Testnet
	Local
	Devnet
	Cluster
)

var (
	DefaultSupportedNetworkOptions = []NetworkOption{
		Local,
		Devnet,
		Testnet,
		Mainnet,
	}
	NonLocalSupportedNetworkOptions = []NetworkOption{
		Devnet,
		Testnet,
		Mainnet,
	}
	NonMainnetSupportedNetworkOptions = []NetworkOption{
		Local,
		Devnet,
		Testnet,
	}
	LocalClusterSupportedNetworkOptions = []NetworkOption{
		Local,
		Cluster,
	}
)

func (n NetworkOption) String() string {
	switch n {
	case Mainnet:
		return "Mainnet"
	case Testnet:
		return "Testnet"
	case Local:
		return "Local Network"
	case Devnet:
		return "Devnet"
	case Cluster:
		return "Cluster"
	}
	return "invalid network"
}

func NetworkOptionFromString(s string) NetworkOption {
	switch {
	case s == "Mainnet":
		return Mainnet
	case s == "Testnet":
		return Testnet
	case s == "Testnet":
		return Testnet
	case s == "Local Network":
		return Local
	case s == "Devnet" || strings.Contains(s, "Devnet"):
		return Devnet
	case s == "Cluster" || strings.Contains(s, "Cluster"):
		return Cluster
	default:
		return Undefined
	}
}

type NetworkFlags struct {
	UseLocal    bool
	UseDevnet   bool
	UseTestnet  bool
	UseMainnet  bool
	Endpoint    string
	ClusterName string
}

func AddNetworkFlagsToCmd(cmd *cobra.Command, networkFlags *NetworkFlags, addEndpoint bool, supportedNetworkOptions []NetworkOption) {
	addCluster := false
	for _, networkOption := range supportedNetworkOptions {
		switch networkOption {
		case Local:
			cmd.Flags().BoolVarP(&networkFlags.UseLocal, "local", "l", false, "operate on a local network")
		case Devnet:
			cmd.Flags().BoolVar(&networkFlags.UseDevnet, "devnet", false, "operate on a devnet network")
			addEndpoint = true
			addCluster = true
		case Testnet:
			cmd.Flags().BoolVarP(&networkFlags.UseTestnet, "testnet", "t", false, "operate on testnet (alias to `testnet`)")
			cmd.Flags().BoolVarP(&networkFlags.UseTestnet, "testnet", "f", false, "operate on testnet (alias to `testnet`")
		case Mainnet:
			cmd.Flags().BoolVarP(&networkFlags.UseMainnet, "mainnet", "m", false, "operate on mainnet")
		case Cluster:
			addCluster = true
		}
	}
	if addCluster {
		cmd.Flags().StringVar(&networkFlags.ClusterName, "cluster", "", "operate on the given cluster")
	}
	if addEndpoint {
		cmd.Flags().StringVar(&networkFlags.Endpoint, "endpoint", "", "use the given endpoint for network operations")
	}
}

func GetNetworkFlagsGroup(cmd *cobra.Command, networkFlags *NetworkFlags, addEndpoint bool, supportedNetworkOptions []NetworkOption) flags.GroupedFlags {
	return flags.RegisterFlagGroup(cmd, "Network Flags (Select One)", "show-network-flags", true, func(set *pflag.FlagSet) {
		addCluster := false
		for _, networkOption := range supportedNetworkOptions {
			switch networkOption {
			case Local:
				set.BoolVarP(&networkFlags.UseLocal, "local", "l", false, "operate on a local network")
			case Devnet:
				set.BoolVar(&networkFlags.UseDevnet, "devnet", false, "operate on a devnet network")
				addEndpoint = true
				addCluster = true
			case Testnet:
				set.BoolVarP(&networkFlags.UseTestnet, "testnet", "t", false, "operate on testnet (alias to `testnet`)")
				set.BoolVarP(&networkFlags.UseTestnet, "testnet", "f", false, "operate on testnet (alias to `testnet`)")
			case Mainnet:
				set.BoolVarP(&networkFlags.UseMainnet, "mainnet", "m", false, "operate on mainnet")
			case Cluster:
				addCluster = true
			}
		}
		if addCluster {
			set.StringVar(&networkFlags.ClusterName, "cluster", "", "operate on the given cluster")
		}
		if addEndpoint {
			set.StringVar(&networkFlags.Endpoint, "endpoint", "", "use the given endpoint for network operations")
		}
	})
}

func GetSupportedNetworkOptionsForSubnet(
	app *application.Lux,
	subnetName string,
	supportedNetworkOptions []NetworkOption,
) ([]NetworkOption, []string, []string, error) {
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return nil, nil, nil, err
	}
	filteredSupportedNetworkOptions := []NetworkOption{}
	for _, networkOption := range supportedNetworkOptions {
		isInSidecar := false
		for networkName := range sc.Networks {
			networkOptionWords := strings.Fields(networkOption.String())
			if len(networkOptionWords) == 0 {
				return nil, nil, nil, fmt.Errorf("empty network option")
			}
			firstNetworkOptionWord := networkOptionWords[0]
			if strings.HasPrefix(networkName, firstNetworkOptionWord) {
				isInSidecar = true
			}
			if os.Getenv(constants.SimulatePublicNetwork) != "" {
				if networkName == Local.String() {
					if networkOption == Testnet || networkOption == Mainnet {
						isInSidecar = true
					}
				}
			}
		}
		if isInSidecar {
			filteredSupportedNetworkOptions = append(filteredSupportedNetworkOptions, networkOption)
		}
	}
	supportsClusters := false
	if _, err := utils.GetIndexInSlice(filteredSupportedNetworkOptions, Cluster); err == nil {
		supportsClusters = true
	}
	supportsDevnets := false
	if _, err := utils.GetIndexInSlice(filteredSupportedNetworkOptions, Devnet); err == nil {
		supportsDevnets = true
	}
	clusterNames := []string{}
	devnetEndpoints := []string{}
	for networkName := range sc.Networks {
		if supportsClusters && strings.HasPrefix(networkName, Cluster.String()) {
			parts := strings.Split(networkName, " ")
			if len(parts) != 2 {
				return nil, nil, nil, fmt.Errorf("expected 'Cluster clusterName' on network name %s", networkName)
			}
			clusterNames = append(clusterNames, parts[1])
		}
		if supportsDevnets && strings.HasPrefix(networkName, Devnet.String()) {
			parts := strings.Split(networkName, " ")
			if len(parts) > 2 {
				return nil, nil, nil, fmt.Errorf("expected 'Devnet endpoint' on network name %s", networkName)
			}
			if len(parts) == 2 {
				endpoint := parts[1]
				devnetEndpoints = append(devnetEndpoints, endpoint)
			}
		}
	}
	return filteredSupportedNetworkOptions, clusterNames, devnetEndpoints, nil
}

func GetNetworkFromSidecar(sc models.Sidecar, defaultOption []NetworkOption) []NetworkOption {
	networkOptionsList := []NetworkOption{}
	for scNetwork := range sc.Networks {
		if NetworkOptionFromString(scNetwork) != Undefined {
			networkOptionsList = append(networkOptionsList, NetworkOptionFromString(scNetwork))
		}
	}

	// default network options to add validator options
	if len(networkOptionsList) == 0 {
		networkOptionsList = defaultOption
	}
	return networkOptionsList
}

func GetNetworkFromCmdLineFlags(
	app *application.Lux,
	promptStr string,
	networkFlags NetworkFlags,
	requireDevnetEndpointSpecification bool,
	onlyEndpointBasedDevnets bool,
	supportedNetworkOptions []NetworkOption,
	subnetName string,
) (models.Network, error) {
	supportedNetworkOptionsToPrompt := supportedNetworkOptions
	if slices.Contains(supportedNetworkOptions, Devnet) && !slices.Contains(supportedNetworkOptions, Cluster) {
		supportedNetworkOptions = append(supportedNetworkOptions, Cluster)
	}
	var err error
	supportedNetworkOptionsStrs := ""
	filteredSupportedNetworkOptionsStrs := ""
	scClusterNames := []string{}
	scDevnetEndpoints := []string{}
	if subnetName != "" {
		var filteredSupportedNetworkOptions []NetworkOption
		filteredSupportedNetworkOptions, scClusterNames, scDevnetEndpoints, err = GetSupportedNetworkOptionsForSubnet(app, subnetName, supportedNetworkOptions)
		if err != nil {
			return models.UndefinedNetwork, err
		}
		supportedNetworkOptionsStrs = strings.Join(sdkutils.Map(supportedNetworkOptions, func(s NetworkOption) string { return s.String() }), ", ")
		filteredSupportedNetworkOptionsStrs = strings.Join(sdkutils.Map(filteredSupportedNetworkOptions, func(s NetworkOption) string { return s.String() }), ", ")
		if len(filteredSupportedNetworkOptions) == 0 {
			return models.UndefinedNetwork, fmt.Errorf("no supported deployed networks available on blockchain %q. please deploy to one of: [%s]", subnetName, supportedNetworkOptionsStrs)
		}
		supportedNetworkOptions = filteredSupportedNetworkOptions
	}
	// supported flags
	networkFlagsMap := map[NetworkOption]string{
		Local:   "--local",
		Devnet:  "--devnet",
		Testnet: "--testnet/--testnet",
		Mainnet: "--mainnet",
		Cluster: "--cluster",
	}
	supportedNetworksFlags := strings.Join(sdkutils.Map(supportedNetworkOptions, func(n NetworkOption) string { return networkFlagsMap[n] }), ", ")
	// received option
	networkOption := Undefined
	switch {
	case networkFlags.UseLocal:
		networkOption = Local
	case networkFlags.UseDevnet:
		networkOption = Devnet
	case networkFlags.UseTestnet:
		networkOption = Testnet
	case networkFlags.UseMainnet:
		networkOption = Mainnet
	case networkFlags.ClusterName != "":
		networkOption = Cluster
	case networkFlags.Endpoint != "":
		switch networkFlags.Endpoint {
		case constants.MainnetAPIEndpoint:
			networkOption = Mainnet
		case constants.TestnetAPIEndpoint:
			networkOption = Testnet
		case constants.LocalAPIEndpoint:
			networkOption = Local
		default:
			networkOption = Devnet
		}
	}
	// unsupported option
	// allow cluster because we can extract underlying network from cluster
	// don't check for unsupported network on e2e run
	if networkOption != Undefined && !slices.Contains(supportedNetworkOptions, networkOption) && networkOption != Cluster && os.Getenv(constants.SimulatePublicNetwork) == "" {
		errMsg := fmt.Errorf("network flag %s is not supported. use one of %s", networkFlagsMap[networkOption], supportedNetworksFlags)
		if subnetName != "" {
			clustersMsg := ""
			endpointsMsg := ""
			if len(scClusterNames) != 0 {
				clustersMsg = fmt.Sprintf(". valid clusters: [%s]", strings.Join(scClusterNames, ", "))
			}
			if len(scDevnetEndpoints) != 0 {
				endpointsMsg = fmt.Sprintf(". valid devnet endpoints: [%s]", strings.Join(scDevnetEndpoints, ", "))
			}
			errMsg = fmt.Errorf("network flag %s is not available on blockchain %s. use one of %s or made a deploy for that network%s%s", networkFlagsMap[networkOption], subnetName, supportedNetworksFlags, clustersMsg, endpointsMsg)
		}
		return models.UndefinedNetwork, errMsg
	}
	// mutual exclusion
	if !flags.EnsureMutuallyExclusive([]bool{networkFlags.UseLocal, networkFlags.UseDevnet, networkFlags.UseTestnet, networkFlags.UseMainnet, networkFlags.ClusterName != ""}) {
		return models.UndefinedNetwork, fmt.Errorf("network flags %s are mutually exclusive", supportedNetworksFlags)
	}

	if networkOption == Undefined {
		if subnetName != "" && supportedNetworkOptionsStrs != filteredSupportedNetworkOptionsStrs {
			ux.Logger.PrintToUser("currently supported deployed networks on %q for this command: [%s]", subnetName, filteredSupportedNetworkOptionsStrs)
			ux.Logger.PrintToUser("for more options, deploy %q to one of: [%s]", subnetName, supportedNetworkOptionsStrs)
			ux.Logger.PrintToUser("")
		}
		// undefined, so prompt
		clusterNames, err := app.ListClusterNames()
		if err != nil {
			return models.UndefinedNetwork, err
		}
		if subnetName != "" {
			clusterNames = scClusterNames
		}
		if len(clusterNames) == 0 {
			if index, err := utils.GetIndexInSlice(supportedNetworkOptionsToPrompt, Cluster); err == nil {
				supportedNetworkOptionsToPrompt = append(supportedNetworkOptionsToPrompt[:index], supportedNetworkOptionsToPrompt[index+1:]...)
			}
		}
		if promptStr == "" {
			promptStr = "Choose a network for the operation"
		}
		networkOptionStr, err := app.Prompt.CaptureList(
			promptStr,
			sdkutils.Map(supportedNetworkOptionsToPrompt, func(n NetworkOption) string { return n.String() }),
		)
		if err != nil {
			return models.UndefinedNetwork, err
		}
		networkOption = NetworkOptionFromString(networkOptionStr)
		if networkOption == Devnet && !onlyEndpointBasedDevnets && len(clusterNames) != 0 {
			endpointOptions := []string{
				"Get Devnet RPC endpoint from an existing node cluster (created from lux node create or lux devnet wiz)",
				"Custom",
			}
			if endpointOption, err := app.Prompt.CaptureList("What is the Devnet rpc Endpoint?", endpointOptions); err != nil {
				return models.UndefinedNetwork, err
			} else if endpointOption == endpointOptions[0] {
				networkOption = Cluster
			}
		}
		if networkOption == Cluster {
			networkFlags.ClusterName, err = app.Prompt.CaptureList(
				"Which cluster would you like to use?",
				clusterNames,
			)
			if err != nil {
				return models.UndefinedNetwork, err
			}
		}
	}

	if networkOption == Devnet && networkFlags.Endpoint == "" && requireDevnetEndpointSpecification {
		if len(scDevnetEndpoints) != 0 {
			networkFlags.Endpoint, err = app.Prompt.CaptureList(
				"Choose an endpoint",
				scDevnetEndpoints,
			)
			if err != nil {
				return models.UndefinedNetwork, err
			}
		} else {
			networkFlags.Endpoint, err = app.Prompt.CaptureURL(fmt.Sprintf("%s Endpoint", networkOption.String()))
			if err != nil {
				return models.UndefinedNetwork, err
			}
		}
	}

	if subnetName != "" && networkFlags.ClusterName != "" {
		if _, err := utils.GetIndexInSlice(scClusterNames, networkFlags.ClusterName); err != nil {
			return models.UndefinedNetwork, fmt.Errorf("blockchain %s has not been deployed to cluster %s", subnetName, networkFlags.ClusterName)
		}
	}

	if networkFlags.Endpoint != "" {
		re := regexp.MustCompile(`/+$`)
		networkFlags.Endpoint = re.ReplaceAllString(networkFlags.Endpoint, "")
	}

	network := models.UndefinedNetwork
	switch networkOption {
	case Local:
		network = models.NewLocalNetwork()
	case Devnet:
		// TODO: Get network ID from devnet endpoint if provided
		// if networkFlags.Endpoint != "" {
		//     infoClient := info.NewClient(networkFlags.Endpoint)
		//     ctx, cancel := utils.GetAPIContext()
		//     defer cancel()
		//     networkID, err = infoClient.GetNetworkID(ctx)
		//     if err != nil {
		//         return models.UndefinedNetwork, err
		//     }
		// }
		network = models.NewDevnetNetwork()
	case Testnet:
		network = models.NewTestnetNetwork()
	case Mainnet:
		network = models.NewMainnetNetwork()
	case Cluster:
		if localnet.LocalClusterExists(app, networkFlags.ClusterName) {
			network, err = localnet.GetLocalClusterNetworkModel(app, networkFlags.ClusterName)
			if err != nil {
				return models.UndefinedNetwork, err
			}
		} else {
			network, err = app.GetClusterNetwork(networkFlags.ClusterName)
			if err != nil {
				return models.UndefinedNetwork, err
			}
		}
	}

	// TODO: Add support for custom endpoints
	// Currently Network is immutable, need to refactor to support custom endpoints
	// if networkFlags.Endpoint != "" {
	//     network.Endpoint = networkFlags.Endpoint
	// }

	return network, nil
}
