package masters

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v3/service/controller/key"
)

// This resource applies the ARM template for the master instances, monitors the process and handles upgrades.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	_, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
