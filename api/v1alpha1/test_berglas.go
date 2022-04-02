//go:build test
package v1alpha1

import (
	"context"
	"fmt"
)

var dummyClient *dummyBerglasClient

type dummyBerglasClient struct {
	data map[string][]byte
}

func (d *dummyBerglasClient) Resolve(ctx context.Context, s string) ([]byte, error) {
	result, ok := d.data[s]
	if !ok {
		return nil, fmt.Errorf("not found: %s", s)
	}
	return result, nil
}

func newBerglasClient(ctx context.Context) (berglasClient, error) {
	return dummyClient, nil
}
