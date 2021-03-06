package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	ProvisioningStateFailed    = "Failed"
	ProvisioningStateSucceeded = "Succeeded"
)

// The goal of scaleUpWorkerVMSSTransition is to double the desired number
// of nodes in worker VMSS in order to provide 1:1 mapping between new
// up-to-date nodes when draining and terminating old nodes.
// This will be done in subsequent reconciliation loops to avoid hitting the
// VMSS api too hard.
func (r *Resource) scaleUpWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// If the old VMSS is still present, we should skip this step.
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if the legacy VMSS %s is still present", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck
	legacyVmss, err := r.getScaleSet(ctx, key.ResourceGroupName(cr), key.LegacyWorkerVMSSName(cr))
	if IsScaleSetNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s is not present", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if legacyVmss != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s is still present", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck

		// The legacy VMSS was found, check the scaling.
		legacyVmssHasInstancesRunning := *legacyVmss.Sku.Capacity > 0
		if legacyVmssHasInstancesRunning {
			// The legacy VMSS has still instances running, skip scaling up.
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s has %d instances: skipping scale up", key.LegacyWorkerVMSSName(cr), *legacyVmss.Sku.Capacity)) // nolint: errcheck
			return WaitNewVMSSWorkers, nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s has 0 instances", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck
	}

	desiredWorkerCount := int64(key.WorkerCount(cr) * 2)
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The desired number of workers is: %d", desiredWorkerCount))

	currentWorkerCount, err := r.getInstancesCount(ctx, cr, key.WorkerVMSSName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The current number of workers is: %d", currentWorkerCount))

	allReady, err := vmsscheck.InstancesAreRunning(ctx, r.logger, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
	if vmsscheck.IsVMSSUnsafeError(err) {
		// VMSS rate limits are not safe, let's wait for next reconciliation loop.
		return ScaleUpWorkerVMSS, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}
	// Not all workers are Running in Azure, wait for next reconciliation loop.
	if !allReady {
		return ScaleUpWorkerVMSS, nil
	}

	// All workers ready, we can scale up if needed.
	if desiredWorkerCount > currentWorkerCount {
		// Raise the worker count by one
		err = r.scaleVMSS(ctx, cr, key.WorkerVMSSName, currentWorkerCount+1)
		if err != nil {
			return "", microerror.Mask(err)
		}

		r.instanceWatchdog.GuardVMSS(ctx, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaled worker VMSS to %d nodes", currentWorkerCount+1))

		// Let's stay in the current state.
		return ScaleUpWorkerVMSS, nil
	}

	// We didn't scale up the VMSS, ready to move to next step.
	return CordonOldWorkers, nil
}

func (r *Resource) getInstancesCount(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string) (int64, error) {
	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return -1, microerror.Mask(err)
	}

	vmss, err := c.Get(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject))
	if err != nil {
		return 0, microerror.Mask(err)
	}

	return *vmss.Sku.Capacity, nil
}

func (r *Resource) getScaleSet(ctx context.Context, resourceGroup string, scaleSetName string) (*compute.VirtualMachineScaleSet, error) {
	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vmss, err := c.Get(ctx, resourceGroup, scaleSetName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &vmss, nil
}

func (r *Resource) scaleDownWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	desiredWorkerCount := int64(key.WorkerCount(cr))

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaling worker VMSS to %d nodes", desiredWorkerCount))

	// Scale down to the desired number of nodes in worker VMSS.
	err = r.scaleVMSS(ctx, cr, key.WorkerVMSSName, desiredWorkerCount)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaled worker VMSS to %d nodes", desiredWorkerCount))

	return DeploymentCompleted, nil
}

func (r *Resource) scaleVMSS(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, nodeCount int64) error {
	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	vmss, err := c.Get(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject))
	if err != nil {
		return microerror.Mask(err)
	}

	*vmss.Sku.Capacity = nodeCount
	res, err := c.CreateOrUpdate(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject), vmss)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.CreateOrUpdateResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
