package setup

import (
	"github.com/giantswarm/apprclient"
	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2e-harness/pkg/framework/resource"
	"github.com/giantswarm/e2e-harness/pkg/release"
	e2eclientsazure "github.com/giantswarm/e2eclients/azure"
	"github.com/giantswarm/e2esetup/k8s"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	namespace       = "giantswarm"
	organization    = "giantswarm"
	quayAddress     = "https://quay.io"
	tillerNamespace = "kube-system"
)

type Config struct {
	AzureClient *e2eclientsazure.Client
	Guest       *framework.Guest
	Host        *framework.Host
	K8s         *k8s.Setup
	Logger      micrologger.Logger
	Release     *release.Release
	Resource    *resource.Resource
}

func NewConfig() (Config, error) {
	var err error

	var azureClient *e2eclientsazure.Client
	{
		azureClient, err = e2eclientsazure.NewClient()
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var logger micrologger.Logger
	{
		c := micrologger.Config{}

		logger, err = micrologger.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var apprClient *apprclient.Client
	{
		c := apprclient.Config{
			Logger: logger,

			Address:      quayAddress,
			Organization: organization,
		}

		apprClient, err = apprclient.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var guest *framework.Guest
	{
		c := framework.GuestConfig{
			Logger: logger,

			ClusterID:    env.ClusterID(),
			CommonDomain: env.CommonDomain(),
		}

		guest, err = framework.NewGuest(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var host *framework.Host
	{
		c := framework.HostConfig{
			Logger: logger,

			ClusterID:  env.ClusterID(),
			VaultToken: env.VaultToken(),
		}

		host, err = framework.NewHost(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var k8sSetup *k8s.Setup
	{
		c := k8s.SetupConfig{
			K8sClient: host.K8sClient(),
			Logger:    logger,
		}

		k8sSetup, err = k8s.NewSetup(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var helmClient *helmclient.Client
	{
		c := helmclient.Config{
			Logger:          logger,
			K8sClient:       host.K8sClient(),
			RestConfig:      host.RestConfig(),
			TillerNamespace: tillerNamespace,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var newRelease *release.Release
	{
		c := release.Config{
			ApprClient: apprClient,
			ExtClient:  host.ExtClient(),
			G8sClient:  host.G8sClient(),
			HelmClient: helmClient,
			K8sClient:  host.K8sClient(),
			Logger:     logger,

			Namespace: namespace,
		}

		newRelease, err = release.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var newResource *resource.Resource
	{
		c := resource.Config{
			ApprClient: apprClient,
			HelmClient: helmClient,
			Logger:     logger,

			Namespace: namespace,
		}

		newResource, err = resource.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	c := Config{
		AzureClient: azureClient,
		Guest:       guest,
		Host:        host,
		K8s:         k8sSetup,
		Logger:      logger,
		Release:     newRelease,
		Resource:    newResource,
	}

	return c, nil
}