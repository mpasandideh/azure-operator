package masters

import (
	"context"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
)

func (r *Resource) emptyStateTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	return CheckFlatcarMigrationNeeded, nil
}
