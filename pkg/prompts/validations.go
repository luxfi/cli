// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package prompts

import (
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/luxfi/cli/v2/pkg/constants"
	"github.com/luxfi/cli/v2/pkg/models"
	"github.com/luxfi/cli/v2/pkg/ux"
	"github.com/luxfi/ids"
	lux_constants "github.com/luxfi/node/v2/v2/utils/constants"
	"github.com/luxfi/node/v2/v2/utils/formatting/address"
	"github.com/luxfi/geth/common"
)

// ValidateURLFormat validates if the provided string is a valid URL
func ValidateURLFormat(input string) error {
	if input == "" {
		return errors.New("URL cannot be empty")
	}
	
	u, err := url.Parse(input)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	if u.Scheme == "" {
		return errors.New("URL must have a scheme (e.g., http:// or https://)")
	}
	
	if u.Host == "" {
		return errors.New("URL must have a host")
	}
	
	return nil
}

func validateEmail(input string) error {
	_, err := mail.ParseAddress(input)
	return err
}

func validatePositiveBigInt(input string) error {
	n := new(big.Int)
	n, ok := n.SetString(input, 10)
	if !ok {
		return errors.New("invalid number")
	}
	if n.Cmp(big.NewInt(0)) == -1 {
		return errors.New("invalid number")
	}
	return nil
}

func validateStakingDuration(input string) error {
	d, err := time.ParseDuration(input)
	if err != nil {
		return err
	}
	if d > constants.MaxStakeDuration {
		return fmt.Errorf("exceeds maximum staking duration of %s", ux.FormatDuration(constants.MaxStakeDuration))
	}
	if d < constants.MinStakeDuration {
		return fmt.Errorf("below the minimum staking duration of %s", ux.FormatDuration(constants.MinStakeDuration))
	}
	return nil
}

func validateTime(input string) error {
	t, err := time.Parse(constants.TimeParseLayout, input)
	if err != nil {
		return err
	}
	if t.Before(time.Now().Add(constants.StakingStartLeadTime)) {
		return fmt.Errorf("time should be at least start from now + %s", constants.StakingStartLeadTime)
	}
	return err
}

func validateNodeID(input string) error {
	_, err := ids.NodeIDFromString(input)
	return err
}

func validateAddress(input string) error {
	if !common.IsHexAddress(input) {
		return errors.New("invalid address")
	}
	return nil
}

func validateExistingFilepath(input string) error {
	if fileInfo, err := os.Stat(input); err == nil && !fileInfo.IsDir() {
		return nil
	}
	return errors.New("file doesn't exist")
}

func validateWeight(input string) error {
	val, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return err
	}
	if val < constants.MinStakeWeight {
		return errors.New("the weight must be an integer between 1 and 100")
	}
	return nil
}

func validateBiggerThanZero(input string) error {
	val, err := strconv.ParseUint(input, 0, 64)
	if err != nil {
		return err
	}
	if val == 0 {
		return errors.New("the value must be bigger than zero")
	}
	return nil
}

func validateURL(input string) error {
	_, err := url.ParseRequestURI(input)
	if err != nil {
		return err
	}
	return nil
}

func validatePChainAddress(input string) (string, error) {
	chainID, hrp, _, err := address.Parse(input)
	if err != nil {
		return "", err
	}

	if chainID != "P" {
		return "", errors.New("this is not a PChain address")
	}
	return hrp, nil
}

func validatePChainTestnetAddress(input string) error {
	hrp, err := validatePChainAddress(input)
	if err != nil {
		return err
	}
	if hrp != lux_constants.TestnetHRP {
		return errors.New("this is not a testnet address")
	}
	return nil
}

func validatePChainMainAddress(input string) error {
	hrp, err := validatePChainAddress(input)
	if err != nil {
		return err
	}
	if hrp != lux_constants.MainnetHRP {
		return errors.New("this is not a mainnet address")
	}
	return nil
}

func validatePChainLocalAddress(input string) error {
	hrp, err := validatePChainAddress(input)
	if err != nil {
		return err
	}
	// ANR uses the `custom` HRP for local networks,
	// but the `local` HRP also exists...
	if hrp != lux_constants.LocalHRP && hrp != lux_constants.FallbackHRP {
		return errors.New("this is not a local nor custom address")
	}
	return nil
}

func getPChainValidationFunc(network models.Network) func(string) error {
	switch network {
	case models.Testnet:
		return validatePChainTestnetAddress
	case models.Mainnet:
		return validatePChainMainAddress
	case models.Local:
		return validatePChainLocalAddress
	default:
		return func(string) error {
			return errors.New("unsupported network")
		}
	}
}

func validateID(input string) error {
	_, err := ids.FromString(input)
	return err
}

func validateNewFilepath(input string) error {
	if _, err := os.Stat(input); err != nil && os.IsNotExist(err) {
		return nil
	}
	return errors.New("file already exists")
}
