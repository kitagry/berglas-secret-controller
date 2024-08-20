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
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
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

	ctx    context.Context
	cancel context.CancelFunc
)

func TestControllers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		CRDInstallOptions: envtest.CRDInstallOptions{
			CleanUpAfterUse: true,
		},
		ControlPlaneStopTimeout: 3 * time.Second,
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())
	cfg.Timeout = 5 * time.Second

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
	}).SetupWithManager(ctx, k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Create BerglasSecret", func() {
	const (
		berglasSecretName      = "berglas-secret"
		berglasSecretNamespace = "default"

		timeout  = time.Second * 10
		interval = time.Millisecond * 10
	)

	Context("When creating BerglasSecret", func() {
		It("Should create Secret", func() {
			By("By creating a new BerglasSecret")
			berglasSecretName := berglasSecretName + "-test1"
			ctx := context.Background()
			berglasSecretLookupKey := types.NamespacedName{Name: berglasSecretName, Namespace: berglasSecretNamespace}
			unlock := setBerglasFunc(func(ctx context.Context, s string) ([]byte, error) {
				Expect(s).Should(Equal("berglas://test/test"))
				return []byte("resolved"), nil
			}, func(ctx context.Context, s string) (string, error) {
				return "version", nil
			})
			defer unlock()
			createdBerglasSecret := createAndCheckBerglasSecret(ctx, CreateBerglasSecretParams{
				NamespacedName: berglasSecretLookupKey,
				BerglasData: map[string]string{
					"test":  "berglas://test/test",
					"test2": "unresolved",
				},
				ExpectSecretData: map[string][]uint8{
					"test":  []uint8("resolved"),
					"test2": []uint8("unresolved"),
				},
				timeout:  timeout,
				interval: interval,
			})

			By("By updating berglasSecret")
			createdBerglasSecret.Spec = batchv1alpha1.BerglasSecretSpec{
				Data: map[string]string{
					"test":  "berglas://test/test",
					"test3": "new secret",
				},
			}
			Expect(k8sClient.Update(ctx, createdBerglasSecret)).Should(Succeed())
			time.Sleep(time.Second * 2)

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

			By("By deleting berglasSecret")
			Expect(k8sClient.Delete(ctx, createdBerglasSecret)).Should(Succeed())
			time.Sleep(time.Second * 2)

			var deletedSecret *v1.Secret
			Eventually(func() bool {
				err := k8sClient.Get(ctx, berglasSecretLookupKey, deletedSecret)
				return err == nil
			}, timeout, interval).Should(BeFalse())
			Expect(deletedSecret).Should(BeNil())
		})
	})

	Context("When same name secret already exists", func() {
		It("Should make berglasSecret status BerglasSecretFailure", func() {
			By("By creating a secret")
			berglasSecretName := berglasSecretName + "-test2"
			ctx := context.Background()
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      berglasSecretName,
					Namespace: berglasSecretNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			time.Sleep(time.Second * 2)

			By("By creating a berglasSecret")
			unlock := setBerglasFunc(func(ctx context.Context, s string) ([]byte, error) {
				return []byte("resolved"), nil
			}, func(ctx context.Context, s string) (string, error) {
				return "version", nil
			})
			defer unlock()
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
			time.Sleep(time.Second * 2)

			By("By getting a berglasSecret")
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

			Expect(createdBerglasSecret.Status.Conditions[len(createdBerglasSecret.Status.Conditions)-1].Type).Should(Equal(batchv1alpha1.BerglasSecretFailure))
		})
	})

	Context("When same name berglassecrets (but other namespace) already exists", func() {
		It("Should create secrets", func() {
			By("By creating berglas secret")
			berglasSecretName := berglasSecretName + "-test3"
			ctx := context.Background()
			berglasSecretLookupKey := types.NamespacedName{Name: berglasSecretName, Namespace: berglasSecretNamespace}
			unlock := setBerglasFunc(func(ctx context.Context, s string) ([]byte, error) {
				return []byte("resolved"), nil
			}, func(ctx context.Context, s string) (string, error) {
				return "version", nil
			})
			defer unlock()
			createAndCheckBerglasSecret(ctx, CreateBerglasSecretParams{
				NamespacedName: berglasSecretLookupKey,
				BerglasData: map[string]string{
					"test":  "berglas://test/test",
					"test2": "unresolved",
				},
				ExpectSecretData: map[string][]uint8{
					"test":  []uint8("resolved"),
					"test2": []uint8("unresolved"),
				},
				timeout:  timeout,
				interval: interval,
			})

			otherNamespace := "other-" + berglasSecretNamespace
			By("creating other namespace")
			otherNamespaceResource := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: otherNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, otherNamespaceResource)).Should(Succeed())

			By("By creating other namespace berglas secret")
			createAndCheckBerglasSecret(ctx, CreateBerglasSecretParams{
				NamespacedName: types.NamespacedName{
					Name:      berglasSecretName,
					Namespace: otherNamespace,
				},
				BerglasData: map[string]string{
					"test2": "unresolved",
					"test3": "berglas://test/test",
				},
				ExpectSecretData: map[string][]uint8{
					"test2": []uint8("unresolved"),
					"test3": []uint8("resolved"),
				},
				timeout:  timeout,
				interval: interval,
			})
		})
	})

	Context("When secret will be changed", func() {
		It("Should refresh BerglasSecret after IntervalRefresh", func() {
			By("By creating a berglasSecret")
			berglasSecretName := berglasSecretName + "-test4"
			ctx := context.Background()
			unlock := setBerglasFunc(func(ctx context.Context, s string) ([]byte, error) {
				return []byte("resolved"), nil
			}, func(ctx context.Context, s string) (string, error) {
				return "version", nil
			})
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
						"test": "berglas://test/test",
					},
					RefreshInterval: toPtr(metav1.Duration{Duration: time.Second * 1}),
				},
			}
			Expect(k8sClient.Create(ctx, berglasSecret)).Should(Succeed())
			unlock()

			By("By getting a berglasSecret")
			unlock = setBerglasFunc(func(ctx context.Context, s string) ([]byte, error) {
				return []byte("resolved2"), nil // updated resolved secret
			}, func(ctx context.Context, s string) (string, error) {
				return "version2", nil // updated version
			})
			defer unlock()
			// wait for refresh interval
			time.Sleep(time.Second * 2)

			berglasSecretLookupKey := types.NamespacedName{Name: berglasSecretName, Namespace: berglasSecretNamespace}
			updatedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, berglasSecretLookupKey, updatedSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(updatedSecret.Data).Should(Equal(map[string][]uint8{
				"test": []uint8("resolved2"),
			}))
		})
	})
})

type CreateBerglasSecretParams struct {
	NamespacedName    types.NamespacedName
	BerglasData       map[string]string
	ExpectSecretData  map[string][]uint8
	timeout, interval time.Duration
}

// create berglasSecret and validate secrets
func createAndCheckBerglasSecret(ctx context.Context, params CreateBerglasSecretParams) *batchv1alpha1.BerglasSecret {
	berglasSecret := &batchv1alpha1.BerglasSecret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch.kitagry.github.io/v1alpha1",
			Kind:       "BerglasSecret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.NamespacedName.Name,
			Namespace: params.NamespacedName.Namespace,
		},
		Spec: batchv1alpha1.BerglasSecretSpec{
			Data: params.BerglasData,
		},
	}
	Expect(k8sClient.Create(ctx, berglasSecret)).Should(Succeed())
	time.Sleep(time.Second * 2)

	createdBerglasSecret := &batchv1alpha1.BerglasSecret{}

	Eventually(func() bool {
		err := k8sClient.Get(ctx, params.NamespacedName, createdBerglasSecret)
		return err == nil
	}, params.timeout, params.interval).Should(BeTrue())
	Expect(createdBerglasSecret.Spec.Data).Should(Equal(params.BerglasData))

	createdSecret := &v1.Secret{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, params.NamespacedName, createdSecret)
		return err == nil
	}, params.timeout, params.interval).Should(BeTrue())
	Expect(createdSecret.Data).Should(Equal(params.ExpectSecretData))

	return createdBerglasSecret
}

type dummyBerglasClient struct{}

var (
	resolveFunc func(ctx context.Context, s string) ([]byte, error) = func(ctx context.Context, s string) ([]byte, error) { return nil, nil }
	versionFunc func(ctx context.Context, s string) (string, error) = func(ctx context.Context, s string) (string, error) { return "", nil }
	mux         sync.Mutex
)

func (*dummyBerglasClient) Resolve(ctx context.Context, s string) ([]byte, error) {
	return resolveFunc(ctx, s)
}

func (*dummyBerglasClient) Version(ctx context.Context, s string) (string, error) {
	return versionFunc(ctx, s)
}

func setBerglasFunc(rFunc func(context.Context, string) ([]byte, error), pFunc func(context.Context, string) (string, error)) func() {
	mux.Lock()
	resolveFunc = rFunc
	versionFunc = pFunc
	return func() {
		mux.Unlock()
	}
}

func toPtr[T any](v T) *T {
	return &v
}
