// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/core"
)

const (
	subnetPrivate = "private"
	subnetPublic  = "public"
)

// Client interface for OCI Clients
type Client interface {
	GetSubnetById(context.Context, string) (*core.Subnet, error)
	GetImageIdByName(ctx context.Context, displayName, compartmentId string) (string, error)
}

// ClientImpl OCI Client implementation
type ClientImpl struct {
	vnClient              core.VirtualNetworkClient
	containerEngineClient containerengine.ContainerEngineClient
}

// NewClient creates a new OCI Client
func NewClient(provider common.ConfigurationProvider) (Client, error) {
	net, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}

	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}

	return &ClientImpl{
		vnClient:              net,
		containerEngineClient: containerEngineClient,
	}, nil
}

// GetImageIdByName retrieves an image OCID given an image name and a compartment id, if that image exists.
func (c *ClientImpl) GetImageIdByName(ctx context.Context, displayName, compartmentId string) (string, error) {
	options, err := c.containerEngineClient.GetNodePoolOptions(ctx, containerengine.GetNodePoolOptionsRequest{
		NodePoolOptionId: common.String("all"),
		CompartmentId:    &compartmentId,
	})
	if err != nil {
		return "", err
	}

	for _, src := range options.Sources {
		if displayName == *src.GetSourceName() {
			return *src.(containerengine.NodeSourceViaImageOption).ImageId, nil
		}
	}
	return "", fmt.Errorf("no images found for %s/%s", compartmentId, displayName)
}

// GetSubnetById retrieves a subnet given that subnet's Id.
func (c *ClientImpl) GetSubnetById(ctx context.Context, subnetId string) (*core.Subnet, error) {
	response, err := c.vnClient.GetSubnet(ctx, core.GetSubnetRequest{
		SubnetId:        &subnetId,
		RequestMetadata: common.RequestMetadata{},
	})
	if err != nil {
		return nil, err
	}

	subnet := response.Subnet
	return &subnet, nil
}

// SubnetAccess returns public or private, depending on a subnet's access type
func SubnetAccess(subnet core.Subnet) string {
	if subnet.ProhibitPublicIpOnVnic != nil && subnet.ProhibitInternetIngress != nil && !*subnet.ProhibitPublicIpOnVnic && !*subnet.ProhibitInternetIngress {
		return subnetPublic
	}
	return subnetPrivate
}
