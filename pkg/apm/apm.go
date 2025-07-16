// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// Placeholder for APM functionality
package apm

import (
	"errors"
)

type Client struct{}

func NewAPM(baseURL string) *Client {
	return &Client{}
}

func (c *Client) GetVM(alias string, version string) (*VMUpload, error) {
	return nil, errors.New("APM not implemented")
}

func (c *Client) AddVM(vm *VMUpload) error {
	return errors.New("APM not implemented")
}