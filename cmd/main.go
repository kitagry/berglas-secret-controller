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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/blendle/zapdriver"
	"github.com/go-logr/zapr"
	"github.com/open-policy-agent/cert-controller/pkg/rotator"

	batchv1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1"
	"github.com/kitagry/berglas-secret-controller/internal/berglas"
	berglascontroller "github.com/kitagry/berglas-secret-controller/internal/controller"
	// +kubebuilder:scaffold:imports
)

const (
	secretName     = "berglas-secret-controller-cert"
	caName         = "berglas-secret-controller-ca"
	caOrganization = "berglas-secret-controller"

	// validatingVwhName is the metadata.name of the BerglasSecretController ValidatingWebhookConfiguration.
	validatingVwhName = "berglas-secret-validating-webhook-configuration"

	// mutatingVwhName is the metadata.name of the BerglasSecretController MutatingWebhookConfiguration. currently, BerglasSecret doen't have a mutating webhook.
	// mutatingVwhName   = "berglas-secret-mutating-webhook-configuration"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = batchv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var certDir string
	var certServiceName string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&certDir, "cert-dir", "/certs", "The directory where certs are stored, defaults to /certs")
	flag.StringVar(&certServiceName, "cert-service-name", "berglas-secret-webhook-service", "The service name used to generate the TLS cert's hostname. Defaults to berglas-secret-webhook-service")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zapConfig := zapdriver.NewProductionConfig()
	logger, err := zapConfig.Build(opts.ZapOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"severity": "ERROR", "message": "unable to create zapdriver: %v"}`, err)
		os.Exit(1)
	}
	ctrl.SetLogger(zapr.NewLogger(logger))

	certSetupFinished := make(chan struct{})
	webhookServer := &waitCertWebhookServer{
		Server: webhook.NewServer(webhook.Options{
			CertDir: certDir,
		}),
		certSetupFinished: certSetupFinished,
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "bbb146c0.kitagry.github.io",
		WebhookServer:    webhookServer,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	berglasClient, err := berglas.New(ctx)
	if err != nil {
		setupLog.Error(err, "failed to create berglas client")
		os.Exit(1)
	}

	if err = (&berglascontroller.BerglasSecretReconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controller").WithName("BerglasSecret"),
		Scheme:  mgr.GetScheme(),
		Berglas: berglasClient,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BerglasSecret")
		os.Exit(1)
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		setupLog.Info("setting up cert rotation")
		webhooks := []rotator.WebhookInfo{
			{
				Name: validatingVwhName,
				Type: rotator.Validating,
			},
			// {
			// 	Name: mutatingVwhName,
			// 	Type: rotator.Mutating,
			// },
		}
		namespace := os.Getenv("POD_NAMESPACE")
		if err := rotator.AddRotator(mgr, &rotator.CertRotator{
			SecretKey: types.NamespacedName{
				Namespace: namespace,
				Name:      secretName,
			},
			CertDir:               certDir,
			CAName:                caName,
			CAOrganization:        caOrganization,
			DNSName:               fmt.Sprintf("%s.%s.svc", certServiceName, namespace),
			IsReady:               certSetupFinished,
			Webhooks:              webhooks,
			RequireLeaderElection: true,
		}); err != nil {
			setupLog.Error(err, "unable to set up cert rotation")
			os.Exit(1)
		}

		// setup webhook manager
		if err = (&batchv1alpha1.BerglasSecret{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "BerglasSecret")
			os.Exit(1)
		}
	} else {
		close(certSetupFinished)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type waitCertWebhookServer struct {
	webhook.Server
	certSetupFinished <-chan struct{}
}

func (w *waitCertWebhookServer) Start(ctx context.Context) error {
	<-w.certSetupFinished

	setupLog.Info("cert setup finished, starting webhook server")
	if err := w.Server.Start(ctx); err != nil {
		return err
	}
	return nil
}
