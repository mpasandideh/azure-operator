package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
)

// This transition function aims at detecting if the master VMSS needs to be migrated from CoreOS to flatcar.
func (r *Resource) waitForBackupConfirmationTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("The reconciliation on the masters resource is stopped until the legacy VMSS instance ETCD backup is completed. When you completed the backup, set the masters's resource status to '%s'", DeallocateLegacyInstance))
	return currentState, nil
}