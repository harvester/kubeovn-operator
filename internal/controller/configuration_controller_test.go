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
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/Masterminds/sprig/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
)

var (
	config      = &kubeovniov1.Configuration{}
	typedConfig = types.NamespacedName{}
)

const (
	newVersion            = "v1.14.1"
	kubeOVNControllerName = "kube-ovn-controller"
)

var _ = Describe("Configuration Controller", func() {
	Context("When reconciling a resource", func() {
		configuration := &kubeovniov1.Configuration{}

		BeforeEach(func() {
			By("creating the webhook secret", func() {
				secret, err := generateWebhookSecret()
				if err != nil {
					testSuiteLogger.Error(err, "webhook-certs")
				}
				Expect(err).ShouldNot(HaveOccurred())
				err = k8sClient.Create(ctx, secret)
				Expect(err).ShouldNot(HaveOccurred())
			})

		})

		AfterEach(func() {
			By("removing webhook secret", func() {
				secret, err := generateWebhookSecret()
				Expect(err).ShouldNot(HaveOccurred())
				err = k8sClient.Delete(ctx, secret)
				Expect(err).ShouldNot(HaveOccurred())
			})

		})

		Describe("reconcile configuration object", Ordered, func() {
			It("create configuration object", func() {
				content, err := os.ReadFile("../../config/samples/kubeovn.io_v1_configuration.yaml")
				Expect(err).ShouldNot(HaveOccurred())
				err = yaml.Unmarshal(content, config)
				Expect(err).Should(BeNil())
				typedConfig = types.NamespacedName{Name: config.GetName(), Namespace: config.GetNamespace()}
				err = k8sClient.Get(ctx, typedConfig, configuration)
				if err != nil && errors.IsNotFound(err) {
					Expect(k8sClient.Create(ctx, config)).To(Succeed())
				}
			})

			It("checking baseline conditions have been set", func() {
				Eventually(func() error {
					resource := &kubeovniov1.Configuration{}
					err := k8sClient.Get(ctx, typedConfig, resource)
					if err != nil {
						return err
					}
					testSuiteLogger.WithValues("current status", resource.Status).Info("current status")
					if len(resource.Status.Conditions) != 5 {
						return fmt.Errorf("expected to find 5 baseline conditions")
					}
					return nil
				}, "30s", "5s").Should(BeNil())
			})

			It("checking master nodes have been discovered from status", func() {
				Eventually(func() error {
					resource := &kubeovniov1.Configuration{}
					err := k8sClient.Get(ctx, typedConfig, resource)
					if err != nil {
						return err
					}
					testSuiteLogger.WithValues("current status", resource.Status).Info("current status")
					if len(resource.Status.MatchingNodeAddresses) == 0 {
						return fmt.Errorf("expected to find at least one master node")
					}
					return nil
				}, "30s", "5s").Should(BeNil())
			})

			It("checking status has been reconcilled to deployed", func() {
				Eventually(func() error {
					resource := &kubeovniov1.Configuration{}
					err := k8sClient.Get(ctx, typedConfig, resource)
					if err != nil {
						return err
					}
					testSuiteLogger.WithValues("current status", resource.Status).Info("current status")
					if resource.Status.Status != kubeovniov1.ConfigurationStatusDeployed {
						return fmt.Errorf("expected to find configuration status to be %s but got %s", kubeovniov1.ConfigurationStatusDeployed, resource.Status.Status)
					}
					return nil
				}, "30s", "5s").Should(BeNil())
			})

			// trigger upgrade
			It("Patch Version to simulate an upgrade", func() {
				cr.Version = newVersion
				// patching is immaterial and is only needed to trigger reconcile of the object
				resource := &kubeovniov1.Configuration{}
				err := k8sClient.Get(ctx, typedConfig, resource)
				Expect(err).ToNot(HaveOccurred())
				resource.Spec.Global.Images.KubeOVNImage.Tag = newVersion
				err = k8sClient.Update(ctx, resource)
				Expect(err).ToNot(HaveOccurred())
			})

			// validate new deployments and daemonsets contain the updated images
			It("checking kube-ovn-controller is using the new image", func() {
				Eventually(func() error {
					d := &appsv1.Deployment{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: kubeOVNControllerName, Namespace: defaultKubeovnNamespace}, d)
					if err != nil {
						return err
					}
					// check image for new tag
					for _, v := range d.Spec.Template.Spec.Containers {
						testSuiteLogger.Info("found image", "tag", v.Image)
						if !strings.Contains(v.Image, newVersion) {
							return fmt.Errorf("waiting for new verion %s to be available in container image", newVersion)
						}
					}
					return nil
				}, "30s", "5s").Should(BeNil())
			})
			// validate node finalizers exist
			It("checking node finalizers exist", func() {
				Eventually(func() error {
					nodeList := corev1.NodeList{}
					err := k8sClient.List(ctx, &nodeList)
					if err != nil {
						return err
					}
					exists := true
					for _, v := range nodeList.Items {
						node := &v
						if !controllerutil.ContainsFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer) {
							testSuiteLogger.WithValues("node", node.GetName()).Info("KubeOVNNodeFinalizer not found")
							exists = exists && false
						}
					}
					if exists {
						return nil
					}
					return fmt.Errorf("waiting for finalizer to be set on all nodes")
				}, "120s", "5s").ShouldNot(HaveOccurred())
			})

			It("Cleanup the specific resource instance Configuration", func() {
				resource := &kubeovniov1.Configuration{}
				err := k8sClient.Get(ctx, typedConfig, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
				Eventually(func() error {
					err := k8sClient.Get(ctx, typedConfig, resource)
					if apierrors.IsNotFound(err) {
						return nil
					}
					return fmt.Errorf("waiting for configuration object to be gc'd")
				}, "30s", "5s").Should(BeNil())
			})

			It("checking node finalizers have been removed", func() {
				nodeList := &corev1.NodeList{}
				Eventually(func() error {
					err := k8sClient.List(ctx, nodeList)
					if err != nil {
						return nil
					}

					var notFound bool
					for _, v := range nodeList.Items {
						node := &v
						if controllerutil.ContainsFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer) {
							notFound = notFound || true
						}
					}

					if !notFound {
						return fmt.Errorf("expected finalisers to have been removed")
					}
					return nil
				}, "30s", "5s").ShouldNot(HaveOccurred())
			})
		})
	})
})

func generateWebhookSecret() (*corev1.Secret, error) {
	var webhookSecretTemplate = `{{- $webhookSvcAltNames := list (printf "kubeovn-operator-webhook-service.%s.svc" "kube-system") (printf "kubeovn-operator-controller-manager-metrics-service.%s.svc" "kube-system") (printf "kube-ovn-webhook.%s.svc" "kube-system")}}
{{- $ca := genCA "kubeovn-operator-ca" 3650 }}
{{- $cert := genSignedCert (printf "kubeovn-operator-webhook-service.%s" "kube-system" )  nil $webhookSvcAltNames 365 $ca }}
apiVersion: v1
kind: Secret
metadata:
  name: webhook-certs
  namespace: "kube-system"
data:
  tls.crt: {{ $cert.Cert | b64enc }}
  tls.key: {{ $cert.Key | b64enc }}
  ca.crt: {{ $ca.Cert | b64enc }}`

	f := sprig.TxtFuncMap()
	tmpl := template.Must(template.New("secret").Funcs(f).Parse(webhookSecretTemplate))
	var output bytes.Buffer
	err := tmpl.Execute(&output, nil)
	if err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	if len(output.Bytes()) == 0 {
		return nil, fmt.Errorf("generated secret is empty")
	}

	secret := &corev1.Secret{}
	return secret, yaml.Unmarshal(output.Bytes(), secret)
}
