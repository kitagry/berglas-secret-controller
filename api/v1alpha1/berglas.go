//go:build !test
package v1alpha1

import (
	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	"golang.org/x/net/context"
)

func newBerglasClient(ctx context.Context) (berglasClient, error) {
	return berglas.New(ctx)
}
