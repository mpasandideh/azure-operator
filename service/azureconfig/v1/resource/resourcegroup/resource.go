package resourcegroup

import (
	"context"
	"time"

	azureresource "github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
)

const (
	// Name is the identifier of the resource.
	Name = "resourcegroupv1"

	deleteTimeout = 30 * time.Minute
	managedBy     = "azure-operator"
)

type Config struct {
	AzureConfig      client.AzureConfig
	Logger           micrologger.Logger
	InstallationName string
}

// Resource manages Azure resource groups.
type Resource struct {
	azureConfig      client.AzureConfig
	logger           micrologger.Logger
	installationName string
}

func New(config Config) (*Resource, error) {
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.InstallationName must not be empty")
	}

	r := &Resource{
		azureConfig:      config.AzureConfig,
		installationName: config.InstallationName,
		logger:           config.Logger,
	}

	return r, nil
}

// GetCurrentState gets the resource group for this cluster from the Azure API.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var group Group
	{
		groupsClient, err := r.getGroupsClient()
		if err != nil {
			return nil, microerror.Mask(err)
		}

		resourceGroup, err := groupsClient.Get(key.ClusterID(customObject))
		if err != nil {
			if client.ResponseWasNotFound(resourceGroup.Response) {
				// Fall through.
				return Group{}, nil
			}

			return nil, microerror.Mask(err)
		}
		group = Group{
			Name:     *resourceGroup.Name,
			Location: *resourceGroup.Location,
			Tags:     to.StringMap(*resourceGroup.Tags),
		}
	}

	return group, nil
}

// GetDesiredState returns the desired resource group for this cluster.
func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resourceGroup := Group{
		Name:     key.ClusterID(customObject),
		Location: key.Location(customObject),
		Tags:     key.ClusterTags(customObject, r.installationName),
	}

	return resourceGroup, nil
}

// NewUpdatePatch returns the patch creating resource group for this cluster if
// it is needed.
func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	patch := framework.NewPatch()

	resourceGroupToCreate, err := r.newCreateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch.SetCreateChange(resourceGroupToCreate)
	return patch, nil
}

// NewDeletePatch returns the patch deleting resource group for this cluster if
// it is needed.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	patch := framework.NewPatch()

	resourceGroupToDelete, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch.SetDeleteChange(resourceGroupToDelete)
	return patch, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// ApplyCreateChange creates the resource group via the Azure API.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createState interface{}) error {
	_, err := toCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "creating Azure resource group")
	}

	resourceGroupToCreate, err := toGroup(createState)
	if err != nil {
		return microerror.Maskf(err, "creating Azure resource group")
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating Azure resource group")

	if resourceGroupToCreate.Name != "" {
		groupClient, err := r.getGroupsClient()
		if err != nil {
			return microerror.Maskf(err, "creating Azure resource group")
		}

		resourceGroup := azureresource.Group{
			Name:      to.StringPtr(resourceGroupToCreate.Name),
			Location:  to.StringPtr(resourceGroupToCreate.Location),
			ManagedBy: to.StringPtr(managedBy),
			Tags:      to.StringMapPtr(resourceGroupToCreate.Tags),
		}
		_, err = groupClient.CreateOrUpdate(resourceGroupToCreate.Name, resourceGroup)
		if err != nil {
			return microerror.Maskf(err, "creating Azure resource group")
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "creating Azure resource group: created")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "creating Azure resource group: already created")
	}

	return nil
}

// ApplyDeleteChange deletes the resource group via the Azure API.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteState interface{}) error {
	_, err := toCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "deleting Azure resource group")
	}
	resourceGroupToDelete, err := toGroup(deleteState)
	if err != nil {
		return microerror.Maskf(err, "deleting Azure resource group")
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "deleting Azure resource group")

	if resourceGroupToDelete.Name != "" {
		groupsClient, err := r.getGroupsClient()
		if err != nil {
			return microerror.Maskf(err, "deleting Azure resource group")
		}

		// Delete the resource group which also deletes all resources it
		// contains. We wait for the error channel while the deletion happens.
		_, errchan := groupsClient.Delete(resourceGroupToDelete.Name, nil)
		select {
		case err := <-errchan:
			if err != nil {
				return microerror.Maskf(err, "deleting Azure resource group")
			}
		case <-time.After(deleteTimeout):
			return microerror.Maskf(timeoutError, "deleting Azure resource group")
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting Azure resource group: deleted")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting Azure resource group: already deleted")
	}

	return nil
}

// ApplyUpdateChange is a noop because resource groups are not updated.
func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateState interface{}) error {
	return nil
}

func (r *Resource) getGroupsClient() (*azureresource.GroupsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.GroupsClient, nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (Group, error) {
	currentResourceGroup, err := toGroup(currentState)
	if err != nil {
		return Group{}, microerror.Mask(err)
	}
	desiredResourceGroup, err := toGroup(desiredState)
	if err != nil {
		return Group{}, microerror.Mask(err)
	}

	if currentResourceGroup.Name == "" {
		return desiredResourceGroup, nil
	}

	return Group{}, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (Group, error) {
	currentResourceGroup, err := toGroup(currentState)
	if err != nil {
		return Group{}, microerror.Mask(err)
	}
	desiredResourceGroup, err := toGroup(desiredState)
	if err != nil {
		return Group{}, microerror.Mask(err)
	}

	if currentResourceGroup.Name != "" {
		return desiredResourceGroup, nil
	}

	return Group{}, nil
}

func toCustomObject(v interface{}) (providerv1alpha1.AzureConfig, error) {
	if v == nil {
		return providerv1alpha1.AzureConfig{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &providerv1alpha1.AzureConfig{}, v)
	}

	customObjectPointer, ok := v.(*providerv1alpha1.AzureConfig)
	if !ok {
		return providerv1alpha1.AzureConfig{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &providerv1alpha1.AzureConfig{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func toGroup(v interface{}) (Group, error) {
	if v == nil {
		return Group{}, nil
	}

	resourceGroup, ok := v.(Group)
	if !ok {
		return Group{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", Group{}, v)
	}

	return resourceGroup, nil
}