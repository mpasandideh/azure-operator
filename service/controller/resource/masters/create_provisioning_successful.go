package masters

import (
	"context"

	"github.com/giantswarm/azure-operator/v3/service/controller/internal/state"
)

func (r *Resource) provisioningSuccessfulTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Master VMs deployment successfully provisioned")
	r.logger.LogCtx(ctx, "level", "debug", "message", "Waiting for API server to come UP")

	return WaitForMastersToBecomeReady, nil
}
