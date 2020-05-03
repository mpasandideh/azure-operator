package etcdstoragesecret

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
)

const (
	Name = "etcdstoragesecret"
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

func (r *Resource) addFileShareURLToContext(ctx context.Context, containerName, storageAccountName, primaryKey string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "setting ETCD containerurl to context")

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	cc.ETCDFileShareURL = fmt.Sprintf("//%s.file.core.windows.net/%s", storageAccountName, containerName)
	cc.ETCDFileShareSecret = primaryKey

	r.logger.LogCtx(ctx, "level", "debug", "message", "set containerurl to context")

	return nil
}

func (r *Resource) getStorageAccountsClient(ctx context.Context) (*storage.AccountsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.StorageAccountsClient, nil
}
