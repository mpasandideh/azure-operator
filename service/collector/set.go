package collector

import (
	"github.com/giantswarm/exporterkit/collector"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

type SetConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	AzureSetting             setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
}

// Set is basically only a wrapper for the operator's collector implementations.
// It eases the iniitialization and prevents some weird import mess so we do not
// have to alias packages.
type Set struct {
	*collector.Set
}

func NewSet(config SetConfig) (*Set, error) {
	var err error

	var deploymentCollector *Deployment
	{
		c := DeploymentConfig{
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			EnvironmentName: config.AzureSetting.EnvironmentName,
		}

		deploymentCollector, err = NewDeployment(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceGroupCollector *ResourceGroup
	{
		c := ResourceGroupConfig{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			EnvironmentName: config.AzureSetting.EnvironmentName,
		}

		resourceGroupCollector, err = NewResourceGroup(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var usageCollector *Usage
	{
		c := UsageConfig{
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			EnvironmentName: config.AzureSetting.EnvironmentName,
			Location:        config.AzureSetting.Location,
		}

		usageCollector, err = NewUsage(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var rateLimitCollector *RateLimit
	{
		c := RateLimitConfig{
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			EnvironmentName:        config.AzureSetting.EnvironmentName,
			Location:               config.AzureSetting.Location,
			CPAzureClientSetConfig: config.HostAzureClientSetConfig,
		}

		rateLimitCollector, err = NewRateLimit(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var vmssRateLimitCollector *VMSSRateLimit
	{
		c := VMSSRateLimitConfig{
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			EnvironmentName:        config.AzureSetting.EnvironmentName,
			Location:               config.AzureSetting.Location,
			CPAzureClientSetConfig: config.HostAzureClientSetConfig,
		}

		vmssRateLimitCollector, err = NewVMSSRateLimit(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var vpnConnectionCollector *VPNConnection
	{
		c := VPNConnectionConfig{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			AzureSetting:             config.AzureSetting,
			HostAzureClientSetConfig: config.HostAzureClientSetConfig,
		}

		vpnConnectionCollector, err = NewVPNConnection(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var collectorSet *collector.Set
	{
		c := collector.SetConfig{
			Collectors: []collector.Interface{
				deploymentCollector,
				resourceGroupCollector,
				usageCollector,
				rateLimitCollector,
				vmssRateLimitCollector,
				vpnConnectionCollector,
			},
			Logger: config.Logger,
		}

		collectorSet, err = collector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Set{
		Set: collectorSet,
	}

	return s, nil
}
