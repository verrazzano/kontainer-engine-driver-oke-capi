// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type Client struct {
	Images  map[string]string
	Subnets map[string]*core.Subnet
}

// GetImageIdByName retrieves an image OCID given an image name and a compartment id, if that image exists.
func (c *Client) GetImageIdByName(ctx context.Context, displayName, compartmentId string) (string, error) {
	imageId, ok := c.Images[displayName]
	if !ok {
		return "", fmt.Errorf("no images found for %s/%s", compartmentId, displayName)
	}
	return imageId, nil
}

// GetSubnetById retrieves a subnet given that subnet's Id.
func (c *Client) GetSubnetById(ctx context.Context, subnetId string) (*core.Subnet, error) {
	subnet, ok := c.Subnets[subnetId]
	if !ok {
		return nil, fmt.Errorf("no subnet found for %s", subnetId)
	}
	return subnet, nil
}
