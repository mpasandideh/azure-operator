package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/service/controller/v4/key"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	endpointsToCreate, err := toEndpoints(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if endpointsToCreate != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "creating Kubernetes endpoints")

		namespace := key.ClusterNamespace(customObject)
		_, err = r.k8sClient.CoreV1().Endpoints(namespace).Create(endpointsToCreate)
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "creating Kubernetes endpoints: created")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "creating Kubernetes endpoints: already created")
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentEndpoints, err := toEndpoints(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredEndpoints, err := toEndpoints(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var endpointsToCreate *corev1.Endpoints
	if currentEndpoints == nil {
		endpointsToCreate = desiredEndpoints
	}

	return endpointsToCreate, nil
}