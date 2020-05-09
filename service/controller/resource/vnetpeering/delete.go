package vnetpeering

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/finalizerskeptcontext"

	"github.com/giantswarm/azure-operator/v3/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cpPeeringClient, err := r.getCPVnetPeeringsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if the vnet peering exists on the control plane vnet")
	_, err = cpPeeringClient.Get(ctx, r.hostVirtualNetworkName, r.hostVirtualNetworkName, key.ResourceGroupName(cr))
	if IsNotFound(err) {
		// This is what we want, all good.
		r.logger.LogCtx(ctx, "level", "debug", "message", "Vnet peering doesn't exist on the control plane vnet")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Vnet peering still exists on the control plane vnet")

	// Keep the finalizer until as long as the peering connection still exists.
	finalizerskeptcontext.SetKept(ctx)

	r.logger.LogCtx(ctx, "level", "debug", "message", "Requesting deletion vnet peering on the control plane vnet")
	_, err = cpPeeringClient.Delete(ctx, r.hostVirtualNetworkName, r.hostVirtualNetworkName, key.ResourceGroupName(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Requested deletion of vnet peering on the control plane vnet")

	return nil
}
