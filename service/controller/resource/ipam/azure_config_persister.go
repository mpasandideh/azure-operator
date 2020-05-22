package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/network"
)

type AzureConfigPersisterConfig struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

type AzureConfigPersister struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

func NewAzureConfigPersister(config AzureConfigPersisterConfig) (*AzureConfigPersister, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	p := &AzureConfigPersister{
		g8sClient: config.G8sClient,
		logger:    config.Logger,
	}

	return p, nil
}

func (p *AzureConfigPersister) Persist(ctx context.Context, vnet net.IPNet, namespace string, name string) error {
	cr, err := p.g8sClient.ProviderV1alpha1().AzureConfigs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	azureNetwork, err := network.Compute(vnet)
	if err != nil {
		return microerror.Mask(err)
	}

	cr.Spec.Azure.VirtualNetwork.CIDR = vnet.String()
	cr.Spec.Azure.VirtualNetwork.CalicoSubnetCIDR = azureNetwork.Calico.String()
	cr.Spec.Azure.VirtualNetwork.MasterSubnetCIDR = azureNetwork.Master.String()
	cr.Spec.Azure.VirtualNetwork.WorkerSubnetCIDR = azureNetwork.Worker.String()

	// TODO ensure if this fields are used or can be removed.
	// NOTE in Azure we disable Calico right now. This is due to a transitioning
	// phase. The k8scloudconfig templates require certain calico valus to be set
	// nonetheless. So we set them here. Later when the Calico setup is
	// straightened out we can improve the handling here.
	cr.Spec.Cluster.Calico.Subnet = azureNetwork.Calico.IP.String()
	cr.Spec.Cluster.Calico.CIDR, _ = azureNetwork.Calico.Mask.Size()

	_, err = p.g8sClient.ProviderV1alpha1().AzureConfigs(namespace).Update(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
