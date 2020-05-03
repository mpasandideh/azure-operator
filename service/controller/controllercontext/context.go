package controllercontext

import (
	"context"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/service/network"
)

type contextKey string

const controllerKey contextKey = "controller"

type Context struct {
	APILBBackendPoolID  string
	AzureClientSet      *client.AzureClientSet
	AzureNetwork        *network.Subnets
	Client              ContextClient
	CloudConfig         cloudconfig.Interface
	ContainerURL        *azblob.ContainerURL
	EtcdLBBackendPoolID string
	ETCDFileShareURL    string
	ETCDFileShareSecret string
	MasterSubnetID      string
	Release             ContextRelease
	WorkerSubnetID      string
}

type ContextRelease struct {
	Components []v1alpha1.ReleaseSpecComponent
}

func (c *Context) Validate() error {
	if c.APILBBackendPoolID == "" {
		return microerror.Maskf(invalidContextError, "%T.APILBBackendPoolID must not be empty", c)
	}
	if c.EtcdLBBackendPoolID == "" {
		return microerror.Maskf(invalidContextError, "%T.EtcdLBBackendPoolID must not be empty", c)
	}
	if c.MasterSubnetID == "" {
		return microerror.Maskf(invalidContextError, "%T.MasterSubnetID must not be empty", c)
	}
	if c.WorkerSubnetID == "" {
		return microerror.Maskf(invalidContextError, "%T.WorkerSubnetID must not be empty", c)
	}

	return nil
}

func NewContext(ctx context.Context, c Context) context.Context {
	return context.WithValue(ctx, controllerKey, &c)
}

func FromContext(ctx context.Context) (*Context, error) {
	c, ok := ctx.Value(controllerKey).(*Context)
	if !ok {
		return nil, microerror.Mask(notFoundError)
	}

	return c, nil
}
