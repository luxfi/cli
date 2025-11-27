// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package prompts

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
	"github.com/manifoldco/promptui"
	"golang.org/x/mod/semver"
)

const (
	Yes = "Yes"
	No  = "No"

	Add        = "Add"
	Del        = "Delete"
	Preview    = "Preview"
	MoreInfo   = "More Info"
	Done       = "Done"
	Cancel     = "Cancel"
	LessThanEq = "Less Than Or Eq"
	MoreThanEq = "More Than Or Eq"
	MoreThan   = "More Than"

	// Address formats
	PChainFormat = "P-Chain"
	CChainFormat = "C-Chain"
)

var errNoKeys = errors.New("no keys")

// promptUIRunner is a variable for testing purposes to allow mocking prompt.Run()
var promptUIRunner = func(prompt promptui.Prompt) (string, error) {
	return prompt.Run()
}

// promptUISelectRunner is a variable for testing purposes to allow mocking select.Run()
var promptUISelectRunner = func(sel promptui.Select) (int, string, error) {
	return sel.Run()
}

// utilsReadLongString is a variable for testing purposes to allow mocking long string reads
var utilsReadLongString = func(msg string, args ...interface{}) (string, error) {
	fmt.Printf(msg, args...)
	reader := bufio.NewReader(os.Stdin)
	longString, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	// Remove newline character at the end
	longString = strings.TrimSuffix(longString, "\n")
	return longString, nil
}

type Comparator struct {
	Label string // Label that identifies reference value
	Type  string // Less Than Eq or More than Eq
	Value uint64 // Value to Compare To
}

func (comparator *Comparator) Validate(val uint64) error {
	switch comparator.Type {
	case LessThanEq:
		if val > comparator.Value {
			return fmt.Errorf("the value must be smaller than or equal to %s (%d)", comparator.Label, comparator.Value)
		}
	case MoreThan:
		if val <= comparator.Value {
			return fmt.Errorf("the value must be bigger than %s (%d)", comparator.Label, comparator.Value)
		}
	case MoreThanEq:
		if val < comparator.Value {
			return fmt.Errorf("the value must be bigger than or equal to %s (%d)", comparator.Label, comparator.Value)
		}
	}
	return nil
}

type Prompter interface {
	CapturePositiveBigInt(promptStr string) (*big.Int, error)
	CaptureAddress(promptStr string) (crypto.Address, error)
	CaptureNewFilepath(promptStr string) (string, error)
	CaptureExistingFilepath(promptStr string) (string, error)
	CaptureYesNo(promptStr string) (bool, error)
	CaptureNoYes(promptStr string) (bool, error)
	CaptureList(promptStr string, options []string) (string, error)
	CaptureString(promptStr string) (string, error)
	CaptureGitURL(promptStr string) (*url.URL, error)
	CaptureURL(promptStr string, validateConnection bool) (string, error)
	CaptureStringAllowEmpty(promptStr string) (string, error)
	CaptureEmail(promptStr string) (string, error)
	CaptureIndex(promptStr string, options []any) (int, error)
	CaptureVersion(promptStr string) (string, error)
	CaptureDuration(promptStr string) (time.Duration, error)
	CaptureDate(promptStr string) (time.Time, error)
	CaptureNodeID(promptStr string) (ids.NodeID, error)
	CaptureID(promptStr string) (ids.ID, error)
	CaptureWeight(promptStr string, validator func(uint64) error) (uint64, error)
	CapturePositiveInt(promptStr string, comparators []Comparator) (int, error)
	CaptureUint64(promptStr string) (uint64, error)
	CaptureUint64Compare(promptStr string, comparators []Comparator) (uint64, error)
	CapturePChainAddress(promptStr string, network models.Network) (string, error)
	CaptureFutureDate(promptStr string, minDate time.Time) (time.Time, error)
	ChooseKeyOrLedger(goal string) (bool, error)
	CaptureValidatorBalance(promptStr string, availableBalance float64, minBalance float64) (float64, error)
	CaptureListWithSize(prompt string, options []string, size int) ([]string, error)
	CaptureFloat(promptStr string, validator func(float64) error) (float64, error)
	CaptureAddresses(promptStr string) ([]crypto.Address, error)
	CaptureXChainAddress(promptStr string, network models.Network) (string, error)
	CaptureValidatedString(promptStr string, validator func(string) error) (string, error)
	CaptureRepoBranch(promptStr string, repo string) (string, error)
	CaptureRepoFile(promptStr string, repo string, branch string) (string, error)
	CaptureInt(promptStr string, validator func(int) error) (int, error)
	CaptureUint8(promptStr string) (uint8, error)
	CaptureFujiDuration(promptStr string) (time.Duration, error)
	CaptureMainnetDuration(promptStr string) (time.Duration, error)
	CaptureMainnetL1StakingDuration(promptStr string) (time.Duration, error)
}

type realPrompter struct{}

// NewProcessChecker creates a new process checker which can respond if the server is running
func NewPrompter() Prompter {
	return &realPrompter{}
}

// CaptureListDecision runs a for loop and continuously asks the
// user for a specific input (currently only `CapturePChainAddress`
// and `CaptureAddress` is supported) until the user cancels or
// chooses `Done`. It does also offer an optional `info` to print
// (if provided) and a preview. Items can also be removed.
func CaptureListDecision[T comparable](
	// we need this in order to be able to run mock tests
	prompter Prompter,
	// the main prompt for entering address keys
	prompt string,
	// the Capture function to use
	capture func(prompt string) (T, error),
	// the prompt for each address
	capturePrompt string,
	// label describes the entity we are prompting for (e.g. address, control key, etc.)
	label string,
	// optional parameter to allow the user to print the info string for more information
	info string,
) ([]T, bool, error) {
	finalList := []T{}
	for {
		listDecision, err := prompter.CaptureList(
			prompt, []string{Add, Del, Preview, MoreInfo, Done, Cancel},
		)
		if err != nil {
			return nil, false, err
		}
		switch listDecision {
		case Add:
			elem, err := capture(capturePrompt)
			if err != nil {
				return nil, false, err
			}
			if contains(finalList, elem) {
				fmt.Println(label + " already in list")
				continue
			}
			finalList = append(finalList, elem)
		case Del:
			if len(finalList) == 0 {
				fmt.Println("No " + label + " added yet")
				continue
			}
			finalListAnyT := []any{}
			for _, v := range finalList {
				finalListAnyT = append(finalListAnyT, v)
			}
			index, err := prompter.CaptureIndex("Choose element to remove:", finalListAnyT)
			if err != nil {
				return nil, false, err
			}
			finalList = append(finalList[:index], finalList[index+1:]...)
		case Preview:
			if len(finalList) == 0 {
				fmt.Println("The list is empty")
				break
			}
			for i, k := range finalList {
				fmt.Printf("%d. %v\n", i, k)
			}
		case MoreInfo:
			if info != "" {
				fmt.Println(info)
			}
		case Done:
			return finalList, false, nil
		case Cancel:
			return nil, true, nil
		default:
			return nil, false, errors.New("unexpected option")
		}
	}
}

func (*realPrompter) CaptureDuration(promptStr string) (time.Duration, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateStakingDuration,
	}

	durationStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return time.ParseDuration(durationStr)
}

func (*realPrompter) CaptureDate(promptStr string) (time.Time, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateTime,
	}

	timeStr, err := promptUIRunner(prompt)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(constants.TimeParseLayout, timeStr)
}

func (*realPrompter) CaptureID(promptStr string) (ids.ID, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateID,
	}

	idStr, err := promptUIRunner(prompt)
	if err != nil {
		return ids.Empty, err
	}
	return ids.FromString(idStr)
}

func (*realPrompter) CaptureNodeID(promptStr string) (ids.NodeID, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateNodeID,
	}

	nodeIDStr, err := promptUIRunner(prompt)
	if err != nil {
		return ids.EmptyNodeID, err
	}
	return ids.NodeIDFromString(nodeIDStr)
}

func (*realPrompter) CaptureWeight(promptStr string, validator func(uint64) error) (uint64, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			// First validate it's a valid weight
			if err := validateWeight(input); err != nil {
				return err
			}
			// Then apply custom validation if provided
			if validator != nil {
				val, err := strconv.ParseUint(input, 10, 64)
				if err != nil {
					return err
				}
				return validator(val)
			}
			return nil
		},
	}

	amountStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(amountStr, 10, 64)
}

func (*realPrompter) CaptureUint64(promptStr string) (uint64, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateBiggerThanZero,
	}

	amountStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(amountStr, 0, 64)
}

func (*realPrompter) CapturePositiveInt(promptStr string, comparators []Comparator) (int, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil {
				return err
			}
			if val < 0 {
				return errors.New("input is less than 0")
			}
			for _, comparator := range comparators {
				if err := comparator.Validate(uint64(val)); err != nil {
					return err
				}
			}
			return nil
		},
	}

	amountStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(amountStr)
}

func (*realPrompter) CaptureUint64Compare(promptStr string, comparators []Comparator) (uint64, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			val, err := strconv.ParseUint(input, 0, 64)
			if err != nil {
				return err
			}
			for _, comparator := range comparators {
				if err := comparator.Validate(val); err != nil {
					return err
				}
			}
			return nil
		},
	}

	amountStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(amountStr, 0, 64)
}

func (*realPrompter) CapturePositiveBigInt(promptStr string) (*big.Int, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validatePositiveBigInt,
	}

	amountStr, err := promptUIRunner(prompt)
	if err != nil {
		return nil, err
	}

	amountInt := new(big.Int)
	amountInt, ok := amountInt.SetString(amountStr, 10)
	if !ok {
		return nil, errors.New("SetString: error")
	}
	return amountInt, nil
}

func (*realPrompter) CapturePChainAddress(promptStr string, network models.Network) (string, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: getPChainValidationFunc(network),
	}

	return promptUIRunner(prompt)
}

func (*realPrompter) CaptureAddress(promptStr string) (crypto.Address, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateAddress,
	}

	addressStr, err := promptUIRunner(prompt)
	if err != nil {
		return crypto.Address{}, err
	}

	// Remove 0x prefix if present
	addr := addressStr
	if len(addressStr) >= 2 && addressStr[0:2] == "0x" {
		addr = addressStr[2:]
	}
	b, _ := hex.DecodeString(addr)
	addressHex := crypto.BytesToAddress(b)
	return addressHex, nil
}

func (*realPrompter) CaptureExistingFilepath(promptStr string) (string, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateExistingFilepath,
	}

	pathStr, err := promptUIRunner(prompt)
	if err != nil {
		return "", err
	}

	return pathStr, nil
}

func (*realPrompter) CaptureNewFilepath(promptStr string) (string, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateNewFilepath,
	}

	pathStr, err := promptUIRunner(prompt)
	if err != nil {
		return "", err
	}

	return pathStr, nil
}

func yesNoBase(promptStr string, orderedOptions []string) (bool, error) {
	prompt := promptui.Select{
		Label: promptStr,
		Items: orderedOptions,
	}

	_, decision, err := promptUISelectRunner(prompt)
	if err != nil {
		return false, err
	}
	return decision == Yes, nil
}

func (*realPrompter) CaptureYesNo(promptStr string) (bool, error) {
	return yesNoBase(promptStr, []string{Yes, No})
}

func (*realPrompter) CaptureNoYes(promptStr string) (bool, error) {
	return yesNoBase(promptStr, []string{No, Yes})
}

func (*realPrompter) CaptureList(promptStr string, options []string) (string, error) {
	prompt := promptui.Select{
		Label: promptStr,
		Items: options,
	}
	_, listDecision, err := promptUISelectRunner(prompt)
	if err != nil {
		return "", err
	}
	return listDecision, nil
}

func (*realPrompter) CaptureEmail(promptStr string) (string, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateEmail,
	}

	str, err := promptUIRunner(prompt)
	if err != nil {
		return "", err
	}

	return str, nil
}

func (*realPrompter) CaptureURL(promptStr string, validateConnection bool) (string, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: ValidateURLFormat,
	}

	urlStr, err := promptUIRunner(prompt)
	if err != nil {
		return "", err
	}

	// Validate connection if requested
	if validateConnection {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}

		// Try to connect to the URL
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Head(urlStr)
		if err != nil {
			// Try GET if HEAD fails
			resp, err = client.Get(urlStr)
			if err != nil {
				return "", fmt.Errorf("failed to connect to %s: %w", parsedURL.Host, err)
			}
		}
		defer resp.Body.Close()

		// Accept any successful response (2xx, 3xx)
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("URL returned error status %d", resp.StatusCode)
		}
	}

	return urlStr, nil
}

func (*realPrompter) CaptureStringAllowEmpty(promptStr string) (string, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
	}

	str, err := promptUIRunner(prompt)
	if err != nil {
		return "", err
	}

	return str, nil
}

func (*realPrompter) CaptureString(promptStr string) (string, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			if input == "" {
				return errors.New("string cannot be empty")
			}
			return nil
		},
	}

	str, err := promptUIRunner(prompt)
	if err != nil {
		return "", err
	}

	return str, nil
}

func (*realPrompter) CaptureGitURL(promptStr string) (*url.URL, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateURL,
	}

	str, err := promptUIRunner(prompt)
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.ParseRequestURI(str)
	if err != nil {
		return nil, err
	}

	return parsedURL, nil
}

func (*realPrompter) CaptureVersion(promptStr string) (string, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			if !semver.IsValid(input) {
				return errors.New("version must be a legal semantic version (ex: v1.1.1)")
			}
			return nil
		},
	}

	str, err := promptUIRunner(prompt)
	if err != nil {
		return "", err
	}

	return str, nil
}

func (*realPrompter) CaptureIndex(promptStr string, options []any) (int, error) {
	prompt := promptui.Select{
		Label: promptStr,
		Items: options,
	}

	listIndex, _, err := promptUISelectRunner(prompt)
	if err != nil {
		return 0, err
	}
	return listIndex, nil
}

// CaptureFutureDate requires from the user a date input which is in the future.
// If `minDate` is not empty, the minimum time in the future from the provided date is required
// Otherwise, time from time.Now() is chosen.
func (*realPrompter) CaptureFutureDate(promptStr string, minDate time.Time) (time.Time, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(s string) error {
			t, err := time.Parse(constants.TimeParseLayout, s)
			if err != nil {
				return err
			}
			if minDate == (time.Time{}) {
				minDate = time.Now()
			}
			if t.Before(minDate.UTC()) {
				return fmt.Errorf("the provided date is before %s UTC", minDate.Format(constants.TimeParseLayout))
			}
			return nil
		},
	}

	timestampStr, err := promptUIRunner(prompt)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(constants.TimeParseLayout, timestampStr)
}

// returns true [resp. false] if user chooses stored key [resp. ledger] option
func (prompter *realPrompter) ChooseKeyOrLedger(goal string) (bool, error) {
	const (
		keyOption    = "Use stored key"
		ledgerOption = "Use ledger"
	)
	option, err := prompter.CaptureList(
		fmt.Sprintf("Which key source should be used to %s?", goal),
		[]string{keyOption, ledgerOption},
	)
	if err != nil {
		return false, err
	}
	return option == keyOption, nil
}

func contains[T comparable](list []T, element T) bool {
	for _, val := range list {
		if val == element {
			return true
		}
	}
	return false
}

// GetKeyOrLedger prompts user to choose between key or ledger
func GetKeyOrLedger(prompter Prompter, goal string, keyDir string, includeEwoq bool) (bool, string, error) {
	useStoredKey, err := prompter.ChooseKeyOrLedger(goal)
	if err != nil {
		return false, "", err
	}
	if !useStoredKey {
		return true, "", nil
	}
	keyName, err := captureKeyName(prompter, goal, keyDir, includeEwoq)
	if err != nil {
		return false, "", err
	}
	return false, keyName, nil
}

func getIndexInSlice[T comparable](list []T, element T) (int, error) {
	for i, val := range list {
		if val == element {
			return i, nil
		}
	}
	return 0, fmt.Errorf("element not found")
}

// check subnet authorization criteria:
// - [subnetAuthKeys] satisfy subnet's [threshold]
// - [subnetAuthKeys] is a subset of subnet's [controlKeys]
func CheckSubnetAuthKeys(subnetAuthKeys []string, controlKeys []string, threshold uint32) error {
	if len(subnetAuthKeys) != int(threshold) {
		return fmt.Errorf("number of given subnet auth differs from the threshold")
	}
	for _, subnetAuthKey := range subnetAuthKeys {
		found := false
		for _, controlKey := range controlKeys {
			if subnetAuthKey == controlKey {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("subnet auth key %s does not belong to control keys", subnetAuthKey)
		}
	}
	return nil
}

// get subnet authorization keys from the user, as a subset of the subnet's [controlKeys]
// with a len equal to the subnet's [threshold]
func GetSubnetAuthKeys(prompt Prompter, controlKeys []string, threshold uint32) ([]string, error) {
	if len(controlKeys) == int(threshold) {
		return controlKeys, nil
	}
	subnetAuthKeys := []string{}
	filteredControlKeys := []string{}
	filteredControlKeys = append(filteredControlKeys, controlKeys...)
	for len(subnetAuthKeys) != int(threshold) {
		subnetAuthKey, err := prompt.CaptureList(
			"Choose a subnet auth key",
			filteredControlKeys,
		)
		if err != nil {
			return nil, err
		}
		index, err := getIndexInSlice(filteredControlKeys, subnetAuthKey)
		if err != nil {
			return nil, err
		}
		subnetAuthKeys = append(subnetAuthKeys, subnetAuthKey)
		filteredControlKeys = append(filteredControlKeys[:index], filteredControlKeys[index+1:]...)
	}
	return subnetAuthKeys, nil
}

func GetTestnetKeyOrLedger(prompt Prompter, goal string, keyDir string) (bool, string, error) {
	useStoredKey, err := prompt.ChooseKeyOrLedger(goal)
	if err != nil {
		return false, "", err
	}
	if !useStoredKey {
		return true, "", nil
	}
	keyName, err := captureKeyName(prompt, goal, keyDir, true) // include ewoq by default
	if err != nil {
		if errors.Is(err, errNoKeys) {
			ux.Logger.PrintToUser("No private keys have been found. Signing transactions on Testnet without a private key " +
				"or ledger is not possible. Create a new one with `lux key create`, or use a ledger device.")
		}
		return false, "", err
	}
	return false, keyName, nil
}

func captureKeyName(prompt Prompter, goal string, keyDir string, includeEwoq bool) (string, error) {
	files, err := os.ReadDir(keyDir)
	if err != nil {
		return "", err
	}

	if len(files) < 1 {
		return "", errNoKeys
	}

	keys := []string{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), constants.KeySuffix) {
			keyName := strings.TrimSuffix(f.Name(), constants.KeySuffix)
			// Skip ewoq key if includeEwoq is false
			if !includeEwoq && keyName == "ewoq" {
				continue
			}
			keys = append(keys, keyName)
		}
	}

	keyName, err := prompt.CaptureList(fmt.Sprintf("Which stored key should be used to %s?", goal), keys)
	if err != nil {
		return "", err
	}

	return keyName, nil
}

func (*realPrompter) CaptureValidatorBalance(promptStr string, availableBalance float64, minBalance float64) (float64, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
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
		},
	}
	result, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(result, 64)
}

// PromptChain prompts the user to select a chain
func PromptChain(
	prompt Prompter,
	message string,
	blockchainNames []string,
	pChainEnabled bool,
	xChainEnabled bool,
	cChainEnabled bool,
	blockchainNameToAvoid string,
	blockchainIDEnabled bool,
) (bool, bool, bool, bool, string, string, error) {
	// Build options
	options := []string{}

	if pChainEnabled {
		options = append(options, "P-Chain")
	}
	if xChainEnabled {
		options = append(options, "X-Chain")
	}
	if cChainEnabled {
		options = append(options, "C-Chain")
	}

	// Add blockchain names
	for _, name := range blockchainNames {
		if name != blockchainNameToAvoid {
			options = append(options, name)
		}
	}

	if blockchainIDEnabled {
		options = append(options, "Enter blockchain ID")
	}

	options = append(options, Cancel)

	choice, err := prompt.CaptureList(message, options)
	if err != nil {
		return false, false, false, false, "", "", err
	}

	if choice == Cancel {
		return true, false, false, false, "", "", nil
	}

	// Return flags based on choice
	pChain := choice == "P-Chain"
	xChain := choice == "X-Chain"
	cChain := choice == "C-Chain"

	blockchainName := ""
	blockchainID := ""

	if choice == "Enter blockchain ID" {
		blockchainID, err = prompt.CaptureString("Enter blockchain ID")
		if err != nil {
			return false, false, false, false, "", "", err
		}
	} else if !pChain && !xChain && !cChain {
		// It's a blockchain name
		blockchainName = choice
	}

	return false, pChain, xChain, cChain, blockchainName, blockchainID, nil
}

// CaptureKeyAddress prompts the user to select a key address
func CaptureKeyAddress(
	prompt Prompter,
	goal string,
	keyDir string,
	getKey func(string) (string, error),
	network models.Network,
	addressFormat string,
) (string, error) {
	// Read available keys from keyDir
	entries, err := os.ReadDir(keyDir)
	if err != nil {
		return "", err
	}

	keys := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pk") {
			keyName := strings.TrimSuffix(entry.Name(), ".pk")
			keys = append(keys, keyName)
		}
	}

	if len(keys) == 0 {
		return "", errNoKeys
	}

	keyName, err := prompt.CaptureList(fmt.Sprintf("Which key should %s?", goal), keys)
	if err != nil {
		return "", err
	}

	keyPath, err := getKey(keyName)
	if err != nil {
		return "", err
	}

	// For now, return the key path
	// In a real implementation, this would convert the key to the appropriate address format
	return keyPath, nil
}

// CaptureListWithSize allows selection of multiple items from a list
func (p realPrompter) CaptureListWithSize(prompt string, options []string, size int) ([]string, error) {
	if len(options) == 0 {
		return nil, errors.New("no options provided")
	}

	selected := []string{}
	remaining := make([]string, len(options))
	copy(remaining, options)

	for i := 0; i < size && len(remaining) > 0; i++ {
		if i > 0 {
			prompt = fmt.Sprintf("Select item %d of %d", i+1, size)
		}

		choice, err := p.CaptureList(prompt, append(remaining, Done))
		if err != nil {
			return nil, err
		}

		if choice == Done {
			break
		}

		selected = append(selected, choice)
		// Remove selected item from remaining options
		newRemaining := []string{}
		for _, opt := range remaining {
			if opt != choice {
				newRemaining = append(newRemaining, opt)
			}
		}
		remaining = newRemaining
	}

	return selected, nil
}

// CaptureFloat prompts the user for a floating point number
func (*realPrompter) CaptureFloat(promptStr string, validator func(float64) error) (float64, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			val, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("strconv.ParseFloat: %v", err)
			}
			if validator != nil {
				return validator(val)
			}
			return nil
		},
	}

	result, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(result, 64)
}

func (*realPrompter) CaptureUint16(promptStr string) (uint16, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			// Support both decimal and hex formats
			base := 10
			numStr := input
			if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
				base = 16
				numStr = input[2:]
			}
			val, err := strconv.ParseUint(numStr, base, 16)
			if err != nil {
				// Include strconv in the error message for tests
				return fmt.Errorf("strconv.ParseUint: %v", err)
			}
			if val > 65535 {
				return errors.New("value must be between 0 and 65535")
			}
			return nil
		},
	}

	result, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	// Parse with proper base detection
	base := 10
	numStr := result
	if strings.HasPrefix(result, "0x") || strings.HasPrefix(result, "0X") {
		base = 16
		numStr = result[2:]
	}
	val, parseErr := strconv.ParseUint(numStr, base, 16)
	if parseErr != nil {
		// Return appropriate error message based on the error type
		if strings.Contains(parseErr.Error(), "value out of range") {
			return 0, errors.New("value out of range")
		}
		return 0, errors.New("invalid syntax")
	}
	return uint16(val), nil
}

func (*realPrompter) CaptureUint32(promptStr string) (uint32, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			// Support both decimal and hex formats
			base := 10
			numStr := input
			if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
				base = 16
				numStr = input[2:]
			}
			_, err := strconv.ParseUint(numStr, base, 32)
			if err != nil {
				// Include strconv in the error message for tests
				return fmt.Errorf("strconv.ParseUint: %v", err)
			}
			return nil
		},
	}

	result, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	// Support both decimal and hex formats for parsing the result
	base := 10
	numStr := result
	if strings.HasPrefix(result, "0x") || strings.HasPrefix(result, "0X") {
		base = 16
		numStr = result[2:]
	}
	val, parseErr := strconv.ParseUint(numStr, base, 32)
	if parseErr != nil {
		// Return appropriate error message based on the error type
		if strings.Contains(parseErr.Error(), "value out of range") {
			return 0, errors.New("value out of range")
		}
		return 0, errors.New("invalid syntax")
	}
	return uint32(val), nil
}

// CaptureAddresses prompts for multiple addresses
func (*realPrompter) CaptureAddresses(promptStr string) ([]crypto.Address, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			// Validate comma-separated addresses
			parts := strings.Split(input, ",")
			for _, part := range parts {
				addr := strings.TrimSpace(part)
				if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
					return fmt.Errorf("invalid address format: %s", addr)
				}
			}
			return nil
		},
	}

	result, err := promptUIRunner(prompt)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(result, ",")
	addresses := make([]crypto.Address, 0, len(parts))
	for _, part := range parts {
		addr := strings.TrimSpace(part)
		addresses = append(addresses, crypto.HexToAddress(addr))
	}

	return addresses, nil
}

// CaptureXChainAddress prompts for an X-Chain address
func (*realPrompter) CaptureXChainAddress(promptStr string, network models.Network) (string, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			if !strings.HasPrefix(input, "X-") && !strings.HasPrefix(input, "x-") {
				return errors.New("X-Chain address must start with X- or x-")
			}
			return nil
		},
	}

	return promptUIRunner(prompt)
}

// CaptureValidatedString prompts for a string with custom validation
func (*realPrompter) CaptureValidatedString(promptStr string, validator func(string) error) (string, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validator,
	}

	return promptUIRunner(prompt)
}

// CaptureRepoBranch prompts for a git branch from a repository
func (*realPrompter) CaptureRepoBranch(promptStr string, repo string) (string, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: ValidateRepoBranch,
	}

	return promptUIRunner(prompt)
}

// CaptureRepoFile prompts for a file path in a repository
func (*realPrompter) CaptureRepoFile(promptStr string, repo string, branch string) (string, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			if input == "" {
				return errors.New("file path cannot be empty")
			}
			// Basic validation for file paths
			if strings.HasPrefix(input, "/") {
				return errors.New("file path should be relative, not absolute")
			}
			return nil
		},
	}

	return promptUIRunner(prompt)
}

// CaptureInt prompts for an integer with custom validation
func (*realPrompter) CaptureInt(promptStr string, validator func(int) error) (int, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("strconv.Atoi: %v", err)
			}
			if validator != nil {
				return validator(val)
			}
			return nil
		},
	}

	result, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(result)
}

// CaptureUint8 prompts for a uint8 value
func (*realPrompter) CaptureUint8(promptStr string) (uint8, error) {
	prompt := promptui.Prompt{
		Label: promptStr,
		Validate: func(input string) error {
			// Support decimal, hex, and octal formats
			base := 10
			numStr := input
			if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
				base = 16
				numStr = input[2:]
			} else if strings.HasPrefix(input, "0") && len(input) > 1 && input != "0" {
				base = 8
				numStr = input[1:]
			}
			_, err := strconv.ParseUint(numStr, base, 8)
			if err != nil {
				return fmt.Errorf("strconv.ParseUint: %v", err)
			}
			return nil
		},
	}

	result, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	// Parse the result with the same logic
	base := 10
	numStr := result
	if strings.HasPrefix(result, "0x") || strings.HasPrefix(result, "0X") {
		base = 16
		numStr = result[2:]
	} else if strings.HasPrefix(result, "0") && len(result) > 1 && result != "0" {
		base = 8
		numStr = result[1:]
	}
	val, err := strconv.ParseUint(numStr, base, 64)
	if err != nil {
		return 0, err
	}
	return uint8(val), nil
}

// CaptureFujiDuration prompts for a staking duration on Fuji testnet
func (*realPrompter) CaptureFujiDuration(promptStr string) (time.Duration, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateTestnetStakingDuration,
	}

	durationStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return time.ParseDuration(durationStr)
}

// CaptureMainnetDuration prompts for a staking duration on mainnet
func (*realPrompter) CaptureMainnetDuration(promptStr string) (time.Duration, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateMainnetStakingDuration,
	}

	durationStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return time.ParseDuration(durationStr)
}

// CaptureMainnetL1StakingDuration prompts for an L1 staking duration on mainnet
func (*realPrompter) CaptureMainnetL1StakingDuration(promptStr string) (time.Duration, error) {
	prompt := promptui.Prompt{
		Label:    promptStr,
		Validate: validateMainnetL1StakingDuration,
	}

	durationStr, err := promptUIRunner(prompt)
	if err != nil {
		return 0, err
	}

	return time.ParseDuration(durationStr)
}
