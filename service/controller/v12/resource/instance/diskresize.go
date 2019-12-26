package instance

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/microerror"
)

const (
	// see template/main.json, 'vmssVmDataDisks' property
	masterDockerVolumeIndex  = 1
	masterKubeletVolumeIndex = 2
	workerDockerVolumeIndex  = 0
	workerKubeletVolumeIndex = 1
)

type diskResizeCheckParams struct {
	Instances               []compute.VirtualMachineScaleSetVM
	DockerVolumeIndex       int
	DockerVolumeSizeGBFunc  func(customObject providerv1alpha1.AzureConfig, index int) int
	KubeletVolumeIndex      int
	KubeletVolumeSizeGBFunc func(customObject providerv1alpha1.AzureConfig, index int) int
	InstanceNameFunc        func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)
}

type diskResizeParams struct {
	instance            *compute.VirtualMachineScaleSetVM
	resizeDockerVolume  bool
	resizeKubeletVolume bool
}

// instanceWithChangedVolumeExists checks if there is a VMSS instance which has
// a disk that has to be resized.
func (r *Resource) instanceWithChangedVolumeExists(ctx context.Context, customObject providerv1alpha1.AzureConfig, checkParams ...diskResizeCheckParams) (bool, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "looking for instance with disk that has to be resized")
	for _, p := range checkParams {
		if p.Instances == nil {
			continue
		}
		for i, v := range p.Instances {
			resizeDockerVolume, resizeKubeletVolume := checkIfVolumesHasChanged(customObject, &v, i, p)
			if resizeDockerVolume || resizeKubeletVolume {
				instanceName, err := p.InstanceNameFunc(customObject, *v.InstanceID)
				if err != nil {
					return false, microerror.Mask(err)
				}
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' with disk that has to be resized", instanceName), "resizeDockerVolume", resizeDockerVolume, "resizeKubeletVolume", resizeKubeletVolume)
				return true, nil
			}
		}
	}

	return false, nil
}

// resizeMasterDisk completes the process of resizing master VMSS instance disk.
// In every reconciliation loop, either one master or one worker VMSS instance
// is resized. The resize steps are the following:
//   - deallocate VMSS instance
//   - start VMSS instance
//   - resize VMSS instance file system
func (r *Resource) resizeMasterDisk(ctx context.Context, customObject providerv1alpha1.AzureConfig, checkParams diskResizeCheckParams) (bool, error) {
	resizeParams, checkedAllInstances, err := r.nextInstanceForDiskResize(ctx, customObject, checkParams)
	if err != nil {
		return false, microerror.Mask(err)
	}
	if resizeParams.instance == nil {
		return true, nil
	}

	c, err := r.getVMsClient(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	instance := resizeParams.instance
	vmssName := key.MasterVMSSName(customObject)
	instanceName, err := key.MasterInstanceName(customObject, *instance.InstanceID)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// step 1: deallocate VMSS instance (shuts down VM and releases attached resources)
	err = r.deallocateVMSSInstance(ctx, customObject, c, instance, vmssName, instanceName)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// step 2: start VMSS instance
	err = r.startVMSSInstance(ctx, customObject, c, instance, vmssName, instanceName)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// step 3: resize VM filesystem
	err = r.resizeVMSSInstanceFilesystem(ctx, customObject, c, resizeParams, vmssName, instanceName)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return checkedAllInstances, nil
}

// nextInstanceForDiskResize finds the next instance which has a disk that has
// to be resized. It check if Docker and/or kubelet volume have changed.
func (r *Resource) nextInstanceForDiskResize(ctx context.Context, customObject providerv1alpha1.AzureConfig, checkParams diskResizeCheckParams) (diskResizeParams, bool, error) {
	resizeParams := diskResizeParams{}
	checkedAllInstances := true
	for i, v := range checkParams.Instances {
		dockerVolumeChanged, kubeletVolumeChanged := checkIfVolumesHasChanged(customObject, &v, i, checkParams)
		if dockerVolumeChanged || kubeletVolumeChanged {
			resizeParams.instance = &v
			resizeParams.resizeDockerVolume = dockerVolumeChanged
			resizeParams.resizeKubeletVolume = kubeletVolumeChanged
			break
		}

		checkedAllInstances = i == len(checkParams.Instances)-1
	}

	if resizeParams.instance != nil {
		instanceName, err := checkParams.InstanceNameFunc(customObject, *resizeParams.instance.InstanceID)
		if err != nil {
			return diskResizeParams{}, checkedAllInstances, microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' with disk that has to be resized", instanceName), "resizeDockerVolume", resizeParams.resizeDockerVolume, "resizeKubeletVolume", resizeParams.resizeKubeletVolume)
	}

	return resizeParams, checkedAllInstances, nil
}

// deallocateVMSSInstance deallocates VMSS instance. It shuts down VM and releases attached resources.
func (r *Resource) deallocateVMSSInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, vmsClient *compute.VirtualMachineScaleSetVMsClient, instance *compute.VirtualMachineScaleSetVM, vmssName string, instanceName string) error {
	resourceGroupName := key.ResourceGroupName(customObject)
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' is deallocated", instanceName))

	future, err := vmsClient.Deallocate(ctx, resourceGroupName, vmssName, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = vmsClient.DeallocateResponder(future.Response())
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' is deallocated", instanceName))

	// check https://github.com/Azure/azure-sdk-for-go/issues/927 for idiomatic way of using futures
	//if err != nil {
	//	return microerror.Mask(err)
	//}
	//err = future.WaitForCompletionRef(ctx, vmsClient.Client)
	//if err != nil {
	//	return microerror.Mask(err)
	//}
	//_, err = future.Result(*vmsClient)
	//if err != nil {
	//	return microerror.Mask(err)
	//}

	return nil
}

// startVMSSInstance starts again VMSS instance.
func (r *Resource) startVMSSInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, vmsClient *compute.VirtualMachineScaleSetVMsClient, instance *compute.VirtualMachineScaleSetVM, vmssName string, instanceName string) error {
	resourceGroupName := key.ResourceGroupName(customObject)
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' is started", instanceName))

	future, err := vmsClient.Start(ctx, resourceGroupName, vmssName, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = vmsClient.StartResponder(future.Response())
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' is started", instanceName))
	return nil
}

// resizeVMSSInstanceFilesystem resizes VMSS instance disk. It does it by
// running following commands on instances:
//   - For Docker volume: sudo xfs_growfs /var/lib/docker
//   - For kubelet volume: sudo xfs_growfs /var/lib/kubelet
func (r *Resource) resizeVMSSInstanceFilesystem(ctx context.Context, customObject providerv1alpha1.AzureConfig, vmsClient *compute.VirtualMachineScaleSetVMsClient, resizeParams diskResizeParams, vmssName string, instanceName string) error {
	resourceGroupName := key.ResourceGroupName(customObject)
	instance := resizeParams.instance
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' filesystem is resized", instanceName))

	commandId := "RunShellScript"
	var script []string
	if resizeParams.resizeDockerVolume {
		script = append(script, "sudo xfs_growfs /var/lib/docker")
	}
	if resizeParams.resizeKubeletVolume {
		script = append(script, "sudo xfs_growfs /var/lib/kubelet")
	}
	commandInput := compute.RunCommandInput{
		CommandID: &commandId,
		Script:    &script,
	}
	future, err := vmsClient.RunCommand(ctx, resourceGroupName, vmssName, *instance.InstanceID, commandInput)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = vmsClient.RunCommandResponder(future.Response())
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' filesystem is resized", instanceName))
	return nil
}

// checkIfVolumesHasChanged checks if Docker volume and/or kubelet volume have
// changed and have to be resized.
func checkIfVolumesHasChanged(customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceIndex int, p diskResizeCheckParams) (dockerVolumeChanged, kubeletVolumeChanged bool) {
	actualDockerVolumeSize := instanceDiskSizeGB(instance, p.DockerVolumeIndex)
	desiredDockerVolumeSize := p.DockerVolumeSizeGBFunc(customObject, instanceIndex)
	if actualDockerVolumeSize != desiredDockerVolumeSize {
		dockerVolumeChanged = true
	}
	actualKubeletVolumeSize := instanceDiskSizeGB(instance, p.KubeletVolumeIndex)
	desiredKubeletVolumeSize := p.KubeletVolumeSizeGBFunc(customObject, instanceIndex)
	if actualKubeletVolumeSize != desiredKubeletVolumeSize {
		kubeletVolumeChanged = true
	}

	return dockerVolumeChanged, kubeletVolumeChanged
}

// instanceDiskSizeGB returns VMSS instance's actual disk size in GB for a disk
// at specified disk index.
func instanceDiskSizeGB(vm *compute.VirtualMachineScaleSetVM, diskIndex int) int {
	dataDisks := *vm.StorageProfile.DataDisks
	return int(*dataDisks[diskIndex].DiskSizeGB)
}
