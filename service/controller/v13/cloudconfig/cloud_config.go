package cloudconfig

import (
	"os"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_5_1_0"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/randomkeys"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/network"
)

const (
	CertFilePermission          = 0400
	CloudProviderFilePermission = 0640
	FileOwnerUserName           = "root"
	FileOwnerGroupName          = "root"
	FileOwnerGroupIDNobody      = 65534
	FilePermission              = 0700
)

type Config struct {
	CertsSearcher      certs.Interface
	Logger             micrologger.Logger
	RandomkeysSearcher randomkeys.Interface

	Azure setting.Azure
	// TODO(pk) remove as soon as we sort calico in Azure provider.
	AzureConfig           client.AzureClientSetConfig
	AzureNetwork          network.Subnets
	IgnitionAdditionPaths []string
	IgnitionBasePath      string
	OIDC                  setting.OIDC
	SSOPublicKey          string
}

type CloudConfig struct {
	logger             micrologger.Logger
	randomkeysSearcher randomkeys.Interface

	azure                 setting.Azure
	azureConfig           client.AzureClientSetConfig
	azureNetwork          network.Subnets
	ignitionAdditionPaths []string
	ignitionBasePath      string
	OIDC                  setting.OIDC
	ssoPublicKey          string
}

func New(config Config) (*CloudConfig, error) {
	for _, p := range config.IgnitionAdditionPaths {
		_, err := os.Stat(p)
		if err != nil {
			return nil, microerror.Maskf(invalidConfigError, "%T.IgnitionAdditionPaths must contain existing directories: %p does not exist", p)
		}
	}
	if config.IgnitionBasePath == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.IgnitionBasePath must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.RandomkeysSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.RandomkeysSearcher must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}

	c := &CloudConfig{
		logger:             config.Logger,
		randomkeysSearcher: config.RandomkeysSearcher,

		azure:                 config.Azure,
		azureConfig:           config.AzureConfig,
		azureNetwork:          config.AzureNetwork,
		ignitionAdditionPaths: config.IgnitionAdditionPaths,
		ignitionBasePath:      config.IgnitionBasePath,
		OIDC:                  config.OIDC,
		ssoPublicKey:          config.SSOPublicKey,
	}

	return c, nil
}

func (c CloudConfig) getEncryptionkey(customObject providerv1alpha1.AzureConfig) (string, error) {
	cluster, err := c.randomkeysSearcher.SearchCluster(key.ClusterID(customObject))
	if err != nil {
		return "", microerror.Mask(err)
	}
	return string(cluster.APIServerEncryptionKey), nil
}

func newCloudConfig(template string, params k8scloudconfig.Params) (string, error) {
	c := k8scloudconfig.DefaultCloudConfigConfig()
	c.Params = params
	c.Template = template

	cloudConfig, err := k8scloudconfig.NewCloudConfig(c)
	if err != nil {
		return "", microerror.Mask(err)
	}
	err = cloudConfig.ExecuteTemplate()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return cloudConfig.String(), nil
}