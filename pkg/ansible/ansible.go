// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package ansible provides utilities for creating and managing Ansible inventories.
package ansible

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/constantsants"
	"github.com/luxfi/sdk/models"
)

// CreateAnsibleHostInventory creates inventory file for ansible
// specifies the ip address of the cloud server and the corresponding ssh cert path for the cloud server
func CreateAnsibleHostInventory(inventoryDirPath, certFilePath, cloudService string, publicIPMap map[string]string, cloudConfigMap models.CloudConfig) error {
	if err := os.MkdirAll(inventoryDirPath, 0o750); err != nil {
		return err
	}
	inventoryHostsFilePath := filepath.Join(inventoryDirPath, constants.AnsibleHostInventoryFileName)
	inventoryFile, err := os.OpenFile(inventoryHostsFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, constants.WriteReadReadPerms) //nolint:gosec // G304: Path is from app's config directory
	if err != nil {
		return err
	}
	defer func() { _ = inventoryFile.Close() }()
	if cloudConfigMap != nil {
		for _, cloudConfig := range cloudConfigMap {
			for _, instanceID := range cloudConfig.InstanceIDs {
				ansibleInstanceID, err := models.HostCloudIDToAnsibleID(cloudService, instanceID)
				if err != nil {
					return err
				}
				if err = writeToInventoryFile(inventoryFile, ansibleInstanceID, publicIPMap[instanceID], cloudConfig.CertFilePath); err != nil {
					return err
				}
			}
		}
	} else {
		for instanceID := range publicIPMap {
			ansibleInstanceID, err := models.HostCloudIDToAnsibleID(cloudService, instanceID)
			if err != nil {
				return err
			}
			if err = writeToInventoryFile(inventoryFile, ansibleInstanceID, publicIPMap[instanceID], certFilePath); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeToInventoryFile(inventoryFile *os.File, ansibleInstanceID, publicIP, certFilePath string) error {
	inventoryContent := ansibleInstanceID
	inventoryContent += " ansible_host="
	inventoryContent += publicIP
	inventoryContent += " ansible_user=ubuntu"
	inventoryContent += fmt.Sprintf(" ansible_ssh_private_key_file=%s", certFilePath)
	inventoryContent += fmt.Sprintf(" ansible_ssh_common_args='%s'", constants.AnsibleSSHUseAgentParams)
	if _, err := inventoryFile.WriteString(inventoryContent + "\n"); err != nil {
		return err
	}
	return nil
}

// WriteNodeConfigsToAnsibleInventory writes node configs to ansible inventory file
func WriteNodeConfigsToAnsibleInventory(inventoryDirPath string, nc []models.NodeConfig) error {
	inventoryHostsFilePath := filepath.Join(inventoryDirPath, constants.AnsibleHostInventoryFileName)
	if err := os.MkdirAll(inventoryDirPath, 0o750); err != nil {
		return err
	}
	inventoryFile, err := os.OpenFile(inventoryHostsFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, constants.WriteReadReadPerms) //nolint:gosec // G304: Path is from app's config directory
	if err != nil {
		return err
	}
	defer func() { _ = inventoryFile.Close() }()
	for _, nodeConfig := range nc {
		nodeID, err := models.HostCloudIDToAnsibleID(nodeConfig.CloudService, nodeConfig.NodeID)
		if err != nil {
			return err
		}
		if err := writeToInventoryFile(inventoryFile, nodeID, nodeConfig.ElasticIP, nodeConfig.CertPath); err != nil {
			return err
		}
	}
	return nil
}

// GetAnsibleHostsFromInventory gets alias of all hosts in an inventory file
func GetAnsibleHostsFromInventory(inventoryDirPath string) ([]string, error) {
	ansibleHostIDs := []string{}
	inventory, err := GetInventoryFromAnsibleInventoryFile(inventoryDirPath)
	if err != nil {
		return nil, err
	}
	for _, host := range inventory {
		ansibleHostIDs = append(ansibleHostIDs, host.NodeID)
	}
	return ansibleHostIDs, nil
}

// GetInventoryFromAnsibleInventoryFile reads hosts from an Ansible inventory file.
func GetInventoryFromAnsibleInventoryFile(inventoryDirPath string) ([]*models.Host, error) {
	inventory := []*models.Host{}
	inventoryHostsFile := filepath.Join(inventoryDirPath, constants.AnsibleHostInventoryFileName)
	file, err := os.Open(inventoryHostsFile) //nolint:gosec // G304: Reading from app's config directory
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// host alias is first element in each line of host inventory file
		parsedHost, err := utils.SplitKeyValueStringToMap(scanner.Text(), " ")
		if err != nil {
			return nil, err
		}
		host := &models.Host{
			NodeID:            strings.Split(scanner.Text(), " ")[0],
			IP:                parsedHost["ansible_host"],
			SSHUser:           parsedHost["ansible_user"],
			SSHPrivateKeyPath: parsedHost["ansible_ssh_private_key_file"],
			SSHCommonArgs:     parsedHost["ansible_ssh_common_args"],
		}
		inventory = append(inventory, host)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return inventory, nil
}

// GetHostByNodeID finds a host by its node ID from the inventory.
func GetHostByNodeID(nodeID string, inventoryDirPath string) (*models.Host, error) {
	allHosts, err := GetInventoryFromAnsibleInventoryFile(inventoryDirPath)
	if err != nil {
		return nil, err
	}
	hosts := utils.Filter(allHosts, func(h *models.Host) bool { return h.NodeID == nodeID })
	switch len(hosts) {
	case 1:
		return hosts[0], nil
	case 0:
		return nil, errors.New("host not found")
	default:
		return nil, errors.New("multiple hosts found")
	}
}

// GetHostMapfromAnsibleInventory returns a map from node ID to host.
func GetHostMapfromAnsibleInventory(inventoryDirPath string) (map[string]*models.Host, error) {
	hostMap := map[string]*models.Host{}
	inventory, err := GetInventoryFromAnsibleInventoryFile(inventoryDirPath)
	if err != nil {
		return nil, err
	}
	for _, host := range inventory {
		hostMap[host.NodeID] = host
	}
	return hostMap, nil
}

// UpdateInventoryHostPublicIP first maps existing ansible inventory host file content
// then it deletes the inventory file and regenerates a new ansible inventory file where it will fetch public IP
// of nodes without elastic IP and update its value in the new ansible inventory file
func UpdateInventoryHostPublicIP(inventoryDirPath string, nodesWithDynamicIP map[string]string) error {
	inventory, err := GetHostMapfromAnsibleInventory(inventoryDirPath)
	if err != nil {
		return err
	}
	inventoryHostsFilePath := filepath.Join(inventoryDirPath, constants.AnsibleHostInventoryFileName)
	if err = os.Remove(inventoryHostsFilePath); err != nil {
		return err
	}
	inventoryFile, err := os.Create(inventoryHostsFilePath) //nolint:gosec // G304: Creating file in app's config directory
	if err != nil {
		return err
	}
	for host, ansibleHostContent := range inventory {
		_, nodeID, err := models.HostAnsibleIDToCloudID(host)
		if err != nil {
			return err
		}
		_, ok := nodesWithDynamicIP[nodeID]
		if !ok {
			if _, err = inventoryFile.WriteString(ansibleHostContent.GetAnsibleInventoryRecord() + "\n"); err != nil {
				return err
			}
		} else {
			ansibleHostContent.IP = nodesWithDynamicIP[nodeID]
			if _, err = inventoryFile.WriteString(ansibleHostContent.GetAnsibleInventoryRecord() + "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}
