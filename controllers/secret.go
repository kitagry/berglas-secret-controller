package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	batchv1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	secretAnnotationKey = "kitagry.github.io/berglasSecret"
	secretVersionKey    = "kitagry.github.io/berglasSecretVersion"
)

func (r *BerglasSecretReconciler) reconcileSecret(ctx context.Context, req ctrl.Request, bs *batchv1alpha1.BerglasSecret) error {
	var secret v1.Secret
	err := r.Get(ctx, req.NamespacedName, &secret)
	if errors.IsNotFound(err) {
		return r.createSecret(ctx, req, bs)
	} else if err != nil {
		return err
	}

	return r.updateSecret(ctx, req, bs, &secret)
}

func (r *BerglasSecretReconciler) createSecret(ctx context.Context, req ctrl.Request, bs *batchv1alpha1.BerglasSecret) error {
	data, err := r.resolveBerglasSchemas(ctx, bs.Spec.Data)
	if err != nil {
		return err
	}

	annotationDataJSON, err := json.Marshal(bs.Spec.Data)
	if err != nil {
		return err
	}

	versionData, err := r.createVersionData(ctx, bs)
	if err != nil {
		return fmt.Errorf("failed to create version data: %w", err)
	}

	versionDataJSON, err := json.Marshal(versionData)
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Annotations: map[string]string{
				secretAnnotationKey: string(annotationDataJSON),
				secretVersionKey:    string(versionDataJSON),
			},
		},
		StringData: data,
	}
	if err := ctrl.SetControllerReference(bs, secret, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, secret); err != nil {
		return err
	}
	return nil
}

func (r *BerglasSecretReconciler) resolveBerglasSchemas(ctx context.Context, data map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(data))
	for key, value := range data {
		ref, err := berglas.ParseReference(value)
		if err != nil {
			result[key] = value
			continue
		}

		plaintext, err := r.Berglas.Resolve(ctx, ref.String())
		if err != nil {
			return nil, err
		}
		result[key] = string(plaintext)
	}
	return result, nil
}

func (r *BerglasSecretReconciler) updateSecret(ctx context.Context, req ctrl.Request, bs *batchv1alpha1.BerglasSecret, secret *v1.Secret) error {
	isChanged, err := r.isChanged(ctx, bs, secret)
	if err != nil {
		return err
	}
	if !isChanged {
		return nil
	}

	// When we update both a berglasSecret and a pod which use the berglasSecret to populate environment variables,
	// the pod might use secret which is not updated yet. So, we delete secret firstly, and then create new secret.
	err = r.Delete(ctx, secret)
	if err != nil {
		return fmt.Errorf("failed to update secret in the step of deleting old secret: %w", err)
	}
	return r.createSecret(ctx, req, bs)
}

func (r *BerglasSecretReconciler) createVersionData(ctx context.Context, bs *batchv1alpha1.BerglasSecret) (map[string]string, error) {
	result := make(map[string]string, len(bs.Spec.Data))
	for key, value := range bs.Spec.Data {
		ref, err := berglas.ParseReference(value)
		if err != nil {
			result[key] = ""
			continue
		}
		v, err := r.Berglas.Version(ctx, ref.String())
		if err != nil {
			return nil, err
		}
		result[key] = v
	}
	return result, nil
}

func (r *BerglasSecretReconciler) isChanged(ctx context.Context, bs *batchv1alpha1.BerglasSecret, secret *v1.Secret) (bool, error) {
	annotationDataStr := secret.Annotations[secretAnnotationKey]
	var annotationData map[string]string
	if err := json.Unmarshal([]byte(annotationDataStr), &annotationData); err != nil {
		return false, fmt.Errorf("failed to get annotation data: %w", err)
	}

	if !maps.Equal(annotationData, bs.Spec.Data) {
		return true, nil
	}

	// This is compatible with the previous version of the controller.
	versionDataStr := secret.Annotations[secretVersionKey]
	if versionDataStr == "" {
		return true, nil
	}

	var versionData map[string]string
	if err := json.Unmarshal([]byte(versionDataStr), &versionData); err != nil {
		return false, fmt.Errorf("failed to get version data: %w", err)
	}

	currentVersionData, err := r.createVersionData(ctx, bs)
	if err != nil {
		return false, err
	}
	for key, value := range currentVersionData {
		if versionData[key] != value {
			return true, nil
		}
	}

	return false, nil
}
