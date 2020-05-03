package etcdstoragesecret

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding ETCD storage account")

	fileShareName := key.ETCDFileShareName()
	groupName := key.ClusterID(cr)
	storageAccountName := key.ETCDStorageAccountName(cr)

	storageAccountsClient, err := r.getStorageAccountsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	keys, err := storageAccountsClient.ListKeys(ctx, groupName, storageAccountName, "")
	if IsStorageAccountNotProvisioned(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account is not provisioned")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)

	r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account")
	err = r.addFileShareURLToContext(ctx, fileShareName, storageAccountName, primaryKey)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
