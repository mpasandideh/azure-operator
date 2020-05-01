package vnetpeering

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
)

const (
	Name = "vnetpeering"
)

type Config struct {
	HostAzureClientSetConfig client.AzureClientSetConfig
	InstallationName         string
	Logger                   micrologger.Logger
	TemplateVersion          string
}

type Resource struct {
	hostAzureClientSetConfig client.AzureClientSetConfig
	installationName         string
	logger                   micrologger.Logger
	templateVersion          string
}

func New(config Config) (*Resource, error) {
	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.HostAzureClientSetConfig.%s", err)
	}

	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	r := &Resource{
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
		installationName:         config.InstallationName,
		logger:                   config.Logger,
		templateVersion:          config.TemplateVersion,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getCPVnetPeeringsClient() (*network.VirtualNetworkPeeringsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VnetPeeringClient, nil
}

func (r *Resource) getVnetClient(ctx context.Context) (*network.VirtualNetworksClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkClient, nil
}
