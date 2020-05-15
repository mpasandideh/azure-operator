package vmssmasters

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v3/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v3/service/controller/internal/state"
)

func (r *Resource) provisioningSuccessfulTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Master VMSS deployment successfully provisioned")
	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if API server is UP")

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		// API server not UP yet, wait.
		r.logger.LogCtx(ctx, "level", "debug", "message", "API server not up yet, waiting.")
		return currentState, nil
	}

	return ClusterUpgradeRequirementCheck, nil
}
