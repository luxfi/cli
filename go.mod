module github.com/luxfi/cli

go 1.25.5

// All dependencies use proper tagged versions for reproducibility

require (
	github.com/aws/aws-sdk-go-v2 v1.41.0
	github.com/aws/aws-sdk-go-v2/config v1.32.6
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.200.0
	github.com/chelnak/ysmrr v0.6.0
	github.com/go-git/go-git/v5 v5.16.2
	github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/luxfi/config v1.1.0
	github.com/luxfi/crypto v1.17.37
	github.com/luxfi/erc20-go v0.2.1
	github.com/luxfi/evm v0.8.28
	github.com/luxfi/geth v1.16.69
	github.com/luxfi/ids v1.2.7
	github.com/luxfi/keychain v1.0.1
	github.com/luxfi/ledger v1.1.6
	github.com/luxfi/log v1.2.5
	github.com/luxfi/lpm v1.7.13
	github.com/luxfi/netrunner v1.15.4
	github.com/luxfi/node v1.22.81
	github.com/luxfi/sdk v1.16.40
	github.com/luxfi/vm v1.0.2
	github.com/luxfi/warp v1.18.2
	github.com/manifoldco/promptui v0.9.0
	github.com/melbahja/goph v1.4.0
	github.com/olekukonko/tablewriter v1.0.9
	github.com/onsi/ginkgo/v2 v2.27.3
	github.com/onsi/gomega v1.38.3
	github.com/otiai10/copy v1.14.1 // indirect
	github.com/pborman/ansi v1.0.0
	github.com/posthog/posthog-go v1.6.1
	github.com/schollz/progressbar/v3 v3.18.0
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/spf13/afero v1.15.0
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.27.1
	golang.org/x/crypto v0.46.0
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/mod v0.31.0
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/oauth2 v0.34.0
	golang.org/x/sync v0.19.0
	golang.org/x/text v0.32.0
	google.golang.org/api v0.247.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect

// Don't replace crate-crypto/go-ipa to avoid verkle compatibility issues
// replace github.com/crate-crypto/go-ipa => github.com/luxfi/crypto/ipa v0.0.1

require (
	github.com/btcsuite/btcd v0.24.2
	github.com/btcsuite/btcd/btcutil v1.1.6
	github.com/dgraph-io/badger/v4 v4.9.0
	github.com/google/go-github/v53 v53.2.0
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674
	github.com/luxfi/constants v1.4.2
	github.com/luxfi/genesis v1.5.19
	github.com/luxfi/go-bip32 v1.0.2
	github.com/luxfi/go-bip39 v1.1.2
	github.com/luxfi/math v1.2.2
	github.com/mattn/go-isatty v0.0.20
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	golang.org/x/sys v0.39.0
	golang.org/x/term v0.38.0
)