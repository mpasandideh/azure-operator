package resourcegroup

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/finalizerskeptcontext"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v4patch1/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v4patch1/key"
)

const (
	// Name is the identifier of the resource.
	Name = "resourcegroupv4patch1"

	managedBy = "azure-operator"
)

type Config struct {
	Logger micrologger.Logger

	Azure            setting.Azure
	InstallationName string
}

// Resource manages Azure resource groups.
type Resource struct {
	logger micrologger.Logger

	azure            setting.Azure
	installationName string
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}

	r := &Resource{
		installationName: config.InstallationName,

		azure:  config.Azure,
		logger: config.Logger,
	}

	return r, nil
}

// EnsureCreated ensures the resource group is created via the Azure API.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring resource group is created")

	resourceGroup := azureresource.Group{
		Name:      to.StringPtr(key.ClusterID(customObject)),
		Location:  to.StringPtr(r.azure.Location),
		ManagedBy: to.StringPtr(managedBy),
		Tags:      key.ClusterTags(customObject, r.installationName),
	}
	_, err = groupsClient.CreateOrUpdate(ctx, *resourceGroup.Name, resourceGroup)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured resource group is created")

	return nil
}

// EnsureDeleted ensures the resource group is deleted via the Azure API.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring resource group deletion")

	_, err = groupsClient.Get(ctx, key.ClusterID(customObject))
	if IsNotFound(err) {
		// fall through
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		_, err := groupsClient.Delete(ctx, key.ClusterID(customObject))
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "resource group deletion in progress")
			finalizerskeptcontext.SetKept(ctx)
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")

			return nil
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured resource group deletion")

	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getGroupsClient(ctx context.Context) (*azureresource.GroupsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.GroupsClient, nil
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