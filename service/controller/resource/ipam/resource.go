package ipam

import (
	"net"
	"reflect"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v4/pkg/locker"
)

const (
	Name = "ipam"
)

const (
	// minAllocatedSubnetMaskBits is the maximum size of guest subnet i.e.
	// smaller number here -> larger subnet per guest cluster. For now anything
	// under 16 doesn't make sense in here.
	minAllocatedSubnetMaskBits = 16
)

type Config struct {
	Checker   Checker
	Collector Collector
	Locker    locker.Interface
	Logger    micrologger.Logger
	Persister Persister

	AllocatedSubnetMaskBits int
	NetworkRange            net.IPNet
}

type Resource struct {
	checker   Checker
	collector Collector
	locker    locker.Interface
	logger    micrologger.Logger
	persister Persister

	allocatedSubnetMask net.IPMask
	networkRange        net.IPNet
}

func New(config Config) (*Resource, error) {
	if config.Checker == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Checker must not be empty", config)
	}
	if config.Collector == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Collector must not be empty", config)
	}
	if config.Locker == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Locker must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Persister == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Persister must not be empty", config)
	}

	if config.AllocatedSubnetMaskBits < minAllocatedSubnetMaskBits {
		return nil, microerror.Maskf(invalidConfigError, "%T.AllocatedSubnetMaskBits (%d) must not be smaller than %d", config, config.AllocatedSubnetMaskBits, minAllocatedSubnetMaskBits)
	}
	if reflect.DeepEqual(config.NetworkRange, net.IPNet{}) {
		return nil, microerror.Maskf(invalidConfigError, "%T.NetworkRange must not be empty", config)
	}

	r := &Resource{
		checker:   config.Checker,
		collector: config.Collector,
		locker:    config.Locker,
		logger:    config.Logger,
		persister: config.Persister,

		allocatedSubnetMask: net.CIDRMask(config.AllocatedSubnetMaskBits, 32),
		networkRange:        config.NetworkRange,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
