package main

import (
	"context"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
)

type berglasClient struct {
	bClient    *berglas.Client
	srManager  *secretmanager.Client
	gcrManager *storage.Client
}

func newBerglasClient(ctx context.Context) (*berglasClient, error) {
	client, err := berglas.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create berglas client: %w", err)
	}

	srManager, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager: %w", err)
	}

	gcrManager, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &berglasClient{
		bClient:    client,
		srManager:  srManager,
		gcrManager: gcrManager,
	}, nil
}

func (b *berglasClient) Resolve(ctx context.Context, s string) ([]byte, error) {
	return b.bClient.Resolve(ctx, s)
}

func (b *berglasClient) Version(ctx context.Context, s string) (string, error) {
	ref, err := berglas.ParseReference(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse reference %s: %w", s, err)
	}

	switch ref.Type() {
	case berglas.ReferenceTypeSecretManager:
		version := ref.Version()
		if version == "" {
			version = "latest"
		}

		v, err := b.srManager.GetSecretVersion(ctx, &secretmanagerpb.GetSecretVersionRequest{
			Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", ref.Project(), ref.Name(), version),
		})
		if err != nil {
			return "", fmt.Errorf("failed to get secret version: %w", err)
		}

		return fmt.Sprintf("%d-%s", v.CreateTime.Seconds, strings.Trim(v.Etag, "\"")), nil
	case berglas.ReferenceTypeStorage:
		obj := b.gcrManager.Bucket(ref.Bucket()).Object(ref.Object())
		attrs, err := obj.Attrs(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get object attributes: %w", err)
		}

		return fmt.Sprintf("%d", attrs.CRC32C), nil
	}
	return "", fmt.Errorf("unknown reference type %v", ref.Type())
}
