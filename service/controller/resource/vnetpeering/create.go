package vnetpeering

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

// This resource applies the ARM template for the worker instances, monitors the process and handles upgrades.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Check if vnet exists.
	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if virtual network exists")
	tcVnetClient, err := r.getVnetClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	tcVnet, err := tcVnetClient.Get(ctx, key.ResourceGroupName(cr), key.VnetName(cr), "")
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Virtual network does not exist")
		reconciliationcanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Virtual network exists")

	peering := r.getVnetPeering(*tcVnet.ID)

	cpPeeringClient, err := r.getCPVnetPeeringsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Ensuring vnet peering exists on the control plane vnet")
	_, err = cpPeeringClient.CreateOrUpdate(ctx, r.installationName, r.installationName, key.ResourceGroupName(cr), peering)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) getVnetPeering(vnetId string) network.VirtualNetworkPeering {
	t := true
	f := false
	peering := network.VirtualNetworkPeering{
		VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: &t,
			AllowForwardedTraffic:     &f,
			AllowGatewayTransit:       &f,
			UseRemoteGateways:         &f,
			RemoteVirtualNetwork: &network.SubResource{
				ID: &vnetId,
			},
		},
	}

	return peering
}
