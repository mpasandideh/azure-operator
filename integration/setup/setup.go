package setup

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/integration/env"
)

// WrapTestMain setup and teardown e2e testing environment.
func WrapTestMain(m *testing.M, c Config) {
	var r int

	ctx := context.Background()

	err := Setup(ctx, c)
	if err != nil {
		log.Printf("%#v\n", err)
		r = 1
	} else {
		r = m.Run()
	}

	if env.KeepResources() != "true" {
		err := Teardown(c)
		if err != nil {
			log.Printf("%#v\n", err)
			r = 1
		}
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(ctx context.Context, c Config) error {
	var err error

	release, err := createGSReleaseContainingOperatorVersion(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = common(ctx, c, *release)
	if err != nil {
		return microerror.Mask(err)
	}

	err = provider(ctx, c, *release)
	if err != nil {
		return microerror.Mask(err)
	}

	err = bastion(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = c.Guest.Setup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
