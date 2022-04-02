//go:build !test

package v1alpha1

import (
	"context"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
)

func newBerglasClient(ctx context.Context) (berglasClient, error) {
	return berglas.New(ctx)
}
