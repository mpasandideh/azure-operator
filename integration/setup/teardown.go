package setup

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/integration/env"
)

// Teardown e2e testing environment.
func Teardown(c Config) error {
	var err error
	ctx := context.Background()
	clusterID := env.ClusterID()

	// Delete AzureConfig from control plane to avoid reconciling while deleting the resources on Azure.
	// We don't wait for the resources to be deleted, since we are going to delete them below.
	err = c.Host.G8sClient().ProviderV1alpha1().AzureConfigs(c.Host.TargetNamespace()).Delete(clusterID, &metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		// Fallthrough. This is fine.
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	// Delete resources on Azure.
	_, err = c.AzureClient.ResourceGroupsClient.Delete(ctx, clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
