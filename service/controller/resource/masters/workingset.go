package masters

import "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"

type workingSet struct {
	instanceToUpdate            *compute.VirtualMachineScaleSetVM
	instanceToDrain             *compute.VirtualMachineScaleSetVM
	instanceToReimage           *compute.VirtualMachineScaleSetVM
	instanceAlreadyBeingUpdated *compute.VirtualMachineScaleSetVM
}

func (ws *workingSet) WithInstanceToUpdate(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceToUpdate = instance
	return ws
}

func (ws *workingSet) WithInstanceToDrain(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceToDrain = instance
	return ws
}

func (ws *workingSet) WithInstanceToReimage(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceToReimage = instance
	return ws
}

func (ws *workingSet) WithInstanceAlreadyBeingUpdated(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceAlreadyBeingUpdated = instance
	return ws
}

func (ws *workingSet) IsWIP() bool {
	if ws == nil {
		return false
	}

	return ws.instanceToUpdate != nil ||
		ws.instanceToDrain != nil ||
		ws.instanceToReimage != nil ||
		ws.instanceAlreadyBeingUpdated != nil
}

func (ws *workingSet) InstanceToUpdate() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToUpdate
}

func (ws *workingSet) InstanceToDrain() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToDrain
}

func (ws *workingSet) InstanceToReimage() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToReimage
}

func (ws *workingSet) InstanceAlreadyBeingUpdated() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceAlreadyBeingUpdated
}
