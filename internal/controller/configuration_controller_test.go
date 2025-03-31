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
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

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
			By("creating the custom resource for the Kind Configuration")
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

		AfterEach(func() {
			resource := &kubeovniov1.Configuration{}
			err := k8sClient.Get(ctx, typedConfig, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Configuration", func() {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
				Eventually(func() error {
					err := k8sClient.Get(ctx, typedConfig, resource)
					if apierrors.IsNotFound(err) {
						return nil
					}
					return fmt.Errorf("waiting for configuration object to be gc'd")
				}, "30s", "5s").Should(BeNil())
			})

			By("checking node finalizers have been removed", func() {
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

		It("reconcile configuration object", func() {
			By("checking baseline conditions have been set", func() {
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

			By("checking master nodes have been discovered from status", func() {
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

			By("checking status has been reconcilled to deployed", func() {
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
			By("Patch Version to simulate an upgrade", func() {
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
			By("checking kube-ovn-controller is using the new image", func() {
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
			// delete a master node to validate cleanup

			// add a new node to ensure ovn db is reconcilled
		})
	})
})
