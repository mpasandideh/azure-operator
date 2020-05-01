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

	// Check if TC vnet exists.
	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if TC virtual network exists")
	tcVnetClient, err := r.getTCVnetClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	tcVnet, err := tcVnetClient.Get(ctx, key.ResourceGroupName(cr), key.VnetName(cr), "")
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "TC Virtual network does not exist")
		reconciliationcanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "TC Virtual network exists")

	// Check if CP vnet exists.
	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if CP virtual network exists")
	cpVnetClient, err := r.getCPVnetClient()
	if err != nil {
		return microerror.Mask(err)
	}

	cpVnet, err := cpVnetClient.Get(ctx, r.installationName, r.installationName, "")
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "CP Virtual network does not exist")
		reconciliationcanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "CP Virtual network exists")

	// Create vnet peering on the tenant cluster side.
	tcPeering := r.getTCVnetPeering(*cpVnet.ID)
	tcPeeringClient, err := r.getTCVnetPeeringsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Ensuring vnet peering exists on the tenant cluster vnet")
	_, err = tcPeeringClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), key.VnetName(cr), r.installationName, tcPeering)
	if err != nil {
		return microerror.Mask(err)
	}

	// Create vnet peering on the control plane side.
	cpPeering := r.getCPVnetPeering(*tcVnet.ID)
	cpPeeringClient, err := r.getCPVnetPeeringsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Ensuring vnet peering exists on the control plane vnet")
	_, err = cpPeeringClient.CreateOrUpdate(ctx, r.installationName, r.installationName, key.ResourceGroupName(cr), cpPeering)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) getCPVnetPeering(vnetId string) network.VirtualNetworkPeering {
	f := false
	peering := network.VirtualNetworkPeering{
		VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: &f,
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

func (r *Resource) getTCVnetPeering(vnetId string) network.VirtualNetworkPeering {
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
