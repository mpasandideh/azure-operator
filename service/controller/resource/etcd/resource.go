package etcd

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v3/service/controller/controllercontext"
)

const (
	Name = "etcd"
)

type Config struct {
	Logger micrologger.Logger
}

type Resource struct {
	logger micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	newResource := &Resource{
		logger: config.Logger,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDNSRecordSetsClient(ctx context.Context) (*dns.RecordSetsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.DNSRecordSetsClient, nil
}

func (r *Resource) getVMSSPrivateIPs(ctx context.Context, resourceGroupName, virtualMachineScaleSetName string) (map[string]string, error) {
	ips := map[string]string{}

	vmsClient, err := r.getVMsClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	netIfClient, err := r.getInterfacesClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vms, err := vmsClient.ListComplete(
		context.Background(),
		resourceGroupName,
		virtualMachineScaleSetName,
		"",
		"",
		"",
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	interfaceMachines := map[string]string{}

	for vms.NotDone() {
		vm := vms.Value()
		hostname := *vm.OsProfile.ComputerName
		for _, networkInterfaceReference := range *vm.NetworkProfile.NetworkInterfaces {
			interfaceMachines[*networkInterfaceReference.ID] = hostname
		}

		err := vms.Next()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	result, err := netIfClient.ListVirtualMachineScaleSetNetworkInterfaces(
		context.Background(),
		resourceGroupName,
		virtualMachineScaleSetName,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for result.NotDone() {
		values := result.Values()
		for _, networkInterface := range values {
			ipConfigurations := *networkInterface.IPConfigurations
			if len(ipConfigurations) != 1 {
				return nil, microerror.Mask(incorrectNumberNetworkInterfacesError)
			}

			ipConfiguration := ipConfigurations[0]
			privateIP := *ipConfiguration.PrivateIPAddress
			if privateIP == "" {
				return nil, microerror.Mask(privateIPAddressEmptyError)
			}

			if machineName, ok := interfaceMachines[*networkInterface.ID]; ok {
				ips[machineName] = privateIP
			}
		}

		err := result.Next()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return ips, nil
}

func (r *Resource) getVMsClient(ctx context.Context) (*compute.VirtualMachineScaleSetVMsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualMachineScaleSetVMsClient, nil
}

func (r *Resource) getInterfacesClient(ctx context.Context) (*network.InterfacesClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.InterfacesClient, nil
}
