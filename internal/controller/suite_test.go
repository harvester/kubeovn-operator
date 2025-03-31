/*
Copyright 2025.

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

package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
	"github.com/harvester/kubeovn-operator/internal/testinfra"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx               context.Context
	cancel            context.CancelFunc
	cluster           testinfra.TestCluster
	cfg               *rest.Config
	k8sClient         client.Client
	crdInstallOptions envtest.CRDInstallOptions
	scheme            = runtime.NewScheme()
	testSuiteLogger   = ctrl.Log.WithName("test suite")
	cr                = &ConfigurationReconciler{}
)

const (
	defaultKubeovnNamespace = "kube-system"
	Version                 = "v1.14.0"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	cluster = testinfra.NewTestCluster()
	var err error
	By("registering various schemas")
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = kubeovniov1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = apiextensionsv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
	err = cluster.CreateCluster(ctx)
	Expect(err).NotTo(HaveOccurred())

	cfg, err = cluster.GetKubeConfig(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("installing CRDs")
	crdInstallOptions = envtest.CRDInstallOptions{
		Scheme:             scheme,
		Paths:              []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfPathMissing: true,
	}
	_, err = envtest.InstallCRDs(cfg, crdInstallOptions)
	Expect(err).NotTo(HaveOccurred())
	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// setup manager for reconcilling objects
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				defaultKubeovnNamespace: {},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())

	cr = &ConfigurationReconciler{
		Client:        mgr.GetClient(),
		Scheme:        scheme,
		Namespace:     defaultKubeovnNamespace,
		EventRecorder: mgr.GetEventRecorderFor("configuration-controller"),
		Log:           logf.FromContext(ctx),
		RestConfig:    mgr.GetConfig(),
		Version:       Version,
	}
	err = cr.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	err = (&NodeReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Namespace:     defaultKubeovnNamespace,
		EventRecorder: mgr.GetEventRecorderFor("node-controller"),
		Log:           logf.FromContext(ctx),
		RestConfig:    mgr.GetConfig(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	err = (&HealthCheckReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		Namespace:           defaultKubeovnNamespace,
		EventRecorder:       mgr.GetEventRecorderFor("health-check-controller"),
		Log:                 logf.FromContext(ctx),
		RestConfig:          mgr.GetConfig(),
		HealthCheckInterval: 300,
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())
	testSuiteLogger.Info("starting manager")
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	time.Sleep(2 * time.Minute)
	By("tearing down the test environment")
	err := envtest.UninstallCRDs(cfg, crdInstallOptions)
	Expect(err).NotTo(HaveOccurred())
	cancel()
	err = cluster.DeleteCluster(ctx)
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k3d")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
