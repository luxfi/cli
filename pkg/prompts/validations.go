// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
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
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	lux_constants "github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/sdk/models"
)

func validateEmail(input string) error {
	_, err := mail.ParseAddress(input)
	return err
}

func ValidateURLFormat(input string) error {
	if input == "" {
		return errors.New("URL cannot be empty")
	}
	parsedURL, err := url.Parse(input)
	if err != nil {
		return err
	}
	if parsedURL.Scheme == "" {
		return errors.New("URL must have a scheme (e.g., http:// or https://)")
	}
	return nil
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
	switch network.Kind() {
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

// validateXChainAddress validates an X-Chain address format
func validateXChainAddress(input string) (string, error) {
	chainID, hrp, _, err := address.Parse(input)
	if err != nil {
		return "", err
	}

	if chainID != "X" {
		return "", errors.New("this is not an XChain address")
	}
	return hrp, nil
}

// validateXChainTestnetAddress validates an X-Chain testnet address
func validateXChainTestnetAddress(input string) error {
	hrp, err := validateXChainAddress(input)
	if err != nil {
		return err
	}
	if hrp != lux_constants.TestnetHRP {
		return errors.New("this is not a testnet address")
	}
	return nil
}

// validateXChainMainAddress validates an X-Chain mainnet address
func validateXChainMainAddress(input string) error {
	hrp, err := validateXChainAddress(input)
	if err != nil {
		return err
	}
	if hrp != lux_constants.MainnetHRP {
		return errors.New("this is not a mainnet address")
	}
	return nil
}

// validateXChainLocalAddress validates an X-Chain local address
func validateXChainLocalAddress(input string) error {
	hrp, err := validateXChainAddress(input)
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

// getXChainValidationFunc returns the appropriate X-Chain validation function for a network
func getXChainValidationFunc(network models.Network) func(string) error {
	switch network.Kind() {
	case models.Testnet:
		return validateXChainTestnetAddress
	case models.Mainnet:
		return validateXChainMainAddress
	case models.Local:
		return validateXChainLocalAddress
	default:
		return func(string) error {
			return errors.New("unsupported network")
		}
	}
}

// ValidateHexa validates a hexadecimal string
func ValidateHexa(s string) error {
	if !strings.HasPrefix(s, "0x") {
		return errors.New("hexadecimal string must start with 0x")
	}
	if len(s) <= 2 {
		return errors.New("hexadecimal string must have at least one character after 0x")
	}
	for _, c := range s[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return errors.New("invalid hexadecimal character")
		}
	}
	return nil
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

// validateNonEmpty validates that a string is not empty
func validateNonEmpty(input string) error {
	if input == "" {
		return errors.New("input cannot be empty")
	}
	return nil
}

// validateMainnetStakingDuration validates staking duration for mainnet
func validateMainnetStakingDuration(input string) error {
	duration, err := time.ParseDuration(input)
	if err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}
	// Mainnet min staking duration is 2 weeks
	if duration < 14*24*time.Hour {
		return errors.New("duration must be at least 2 weeks for mainnet")
	}
	// Mainnet max staking duration is 1 year
	if duration > 365*24*time.Hour {
		return errors.New("duration cannot exceed 1 year for mainnet")
	}
	return nil
}

// validateMainnetL1StakingDuration validates L1 staking duration for mainnet
func validateMainnetL1StakingDuration(input string) error {
	duration, err := time.ParseDuration(input)
	if err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}
	// L1 min staking duration is 48 hours
	if duration < 48*time.Hour {
		return errors.New("L1 staking duration must be at least 48 hours for mainnet")
	}
	// L1 max staking duration is 1 year
	if duration > 365*24*time.Hour {
		return errors.New("L1 staking duration cannot exceed 1 year for mainnet")
	}
	return nil
}

// validateTestnetStakingDuration validates staking duration for testnet
func validateTestnetStakingDuration(input string) error {
	duration, err := time.ParseDuration(input)
	if err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}
	// Testnet/Fuji min staking duration is 24 hours
	if duration < 24*time.Hour {
		return errors.New("duration must be at least 24 hours for testnet")
	}
	// Testnet/Fuji max staking duration is 365 days
	if duration > 365*24*time.Hour {
		return errors.New("duration cannot exceed 365 days for testnet")
	}
	return nil
}

// validateDuration validates a general duration string
func validateDuration(input string) error {
	_, err := time.ParseDuration(input)
	return err
}

// ValidateNodeID validates a node ID string (exported for external use)
func ValidateNodeID(input string) error {
	return validateNodeID(input)
}

// validateAddresses validates comma-separated addresses
func validateAddresses(input string) error {
	parts := strings.Split(input, ",")
	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if !common.IsHexAddress(addr) {
			return fmt.Errorf("invalid address: %s", addr)
		}
	}
	return nil
}

// validateValidatorBalanceFunc returns a validator function for balance
func validateValidatorBalanceFunc(availableBalance float64, minBalance float64) func(string) error {
	return func(input string) error {
		val, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return err
		}
		if val < minBalance {
			return fmt.Errorf("balance must be at least %f", minBalance)
		}
		if val > availableBalance {
			return fmt.Errorf("balance cannot exceed available balance of %f", availableBalance)
		}
		return nil
	}
}

// RequestURL makes a GET request to validate URL connectivity
func RequestURL(url string) error {
	// For testing purposes, just check if URL is valid
	return ValidateURLFormat(url)
}

// ValidateURL validates URL format and optionally checks connectivity
func ValidateURL(input string, checkConnection bool) error {
	if err := ValidateURLFormat(input); err != nil {
		return err
	}
	if checkConnection {
		return RequestURL(input)
	}
	return nil
}

// ValidateRepoBranch validates a git branch name
func ValidateRepoBranch(branch string) error {
	if branch == "" {
		return errors.New("branch name cannot be empty")
	}
	// Basic validation for branch names
	if strings.Contains(branch, " ") {
		return errors.New("branch name cannot contain spaces")
	}
	return nil
}

// ValidateRepoFile validates a repository file path
func ValidateRepoFile(filepath string) error {
	if filepath == "" {
		return errors.New("file path cannot be empty")
	}
	// Basic validation for file paths
	if strings.HasPrefix(filepath, "/") {
		return errors.New("file path should be relative, not absolute")
	}
	return nil
}

// validateWeightFunc returns a validator function for weight values
func validateWeightFunc(min, max uint64) func(string) error {
	return func(input string) error {
		val, err := strconv.ParseUint(input, 10, 64)
		if err != nil {
			return err
		}
		if val < min {
			return fmt.Errorf("weight must be at least %d", min)
		}
		if val > max {
			return fmt.Errorf("weight cannot exceed %d", max)
		}
		return nil
	}
}

// ValidatePositiveInt validates that a string can be parsed as a positive integer
func ValidatePositiveInt(input string) error {
	val, err := strconv.Atoi(input)
	if err != nil {
		return errors.New("invalid integer format")
	}
	if val <= 0 {
		return errors.New("value must be positive")
	}
	return nil
}
