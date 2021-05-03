/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	batchv1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = batchv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	berglasClient := &dummyBerglasClient{}
	err = (&BerglasSecretReconciler{
		Client:  k8sManager.GetClient(),
		Log:     k8sManager.GetLogger(),
		Scheme:  k8sManager.GetScheme(),
		Berglas: berglasClient,
	}).SetupWithManager(context.Background(), k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Create BerglasSecret", func() {
	const (
		berglasSecretName      = "berglas-secret"
		berglasSecretNamespace = "default"

		timeout = time.Second * 10
		// duration = time.Second * 10
		interval = time.Millisecond * 10
	)

	Context("When creating BerglasSecret", func() {
		It("Should create Secret", func() {
			By("By creating a new BerglasSecret")
			ctx := context.Background()
			berglasSecret := &batchv1alpha1.BerglasSecret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "batch.kitagry.github.io/v1alpha1",
					Kind:       "BerglasSecret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      berglasSecretName,
					Namespace: berglasSecretNamespace,
				},
				Spec: batchv1alpha1.BerglasSecretSpec{
					Data: map[string]string{
						"test":  "berglas://test/test",
						"test2": "unresolved",
					},
				},
			}
			Expect(k8sClient.Create(ctx, berglasSecret)).Should(Succeed())
			time.Sleep(time.Second * 5)

			berglasSecretLookupKey := types.NamespacedName{Name: berglasSecretName, Namespace: berglasSecretNamespace}
			createdBerglasSecret := &batchv1alpha1.BerglasSecret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, berglasSecretLookupKey, createdBerglasSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdBerglasSecret.Spec.Data).Should(Equal(map[string]string{
				"test":  "berglas://test/test",
				"test2": "unresolved",
			}))

			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, berglasSecretLookupKey, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdSecret.Data).Should(Equal(map[string][]uint8{
				"test":  []uint8("resolved"),
				"test2": []uint8("unresolved"),
			}))

			By("By updating berglasSecret")
			createdBerglasSecret.Spec = batchv1alpha1.BerglasSecretSpec{
				Data: map[string]string{
					"test":  "berglas://test/test",
					"test3": "new secret",
				},
			}
			Expect(k8sClient.Update(ctx, createdBerglasSecret)).Should(Succeed())

			updatedBerglasSecret := &batchv1alpha1.BerglasSecret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, berglasSecretLookupKey, updatedBerglasSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(updatedBerglasSecret.Spec.Data).Should(Equal(map[string]string{
				"test":  "berglas://test/test",
				"test3": "new secret",
			}))

			updatedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, berglasSecretLookupKey, updatedSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(updatedSecret.Data).Should(Equal(map[string][]uint8{
				"test":  []uint8("resolved"),
				"test3": []uint8("new secret"),
			}))
		})
	})
})

type dummyBerglasClient struct{}

func (*dummyBerglasClient) Resolve(ctx context.Context, s string) ([]byte, error) {
	return []byte("resolved"), nil
}
