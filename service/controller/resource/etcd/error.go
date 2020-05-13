package etcd

import (
	"github.com/giantswarm/microerror"
)

var incorrectNumberNetworkInterfacesError = &microerror.Error{
	Kind: "incorrectNumberNetworkInterfacesError",
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var privateIPAddressEmptyError = &microerror.Error{
	Kind: "privateIPAddressEmptyError",
}
