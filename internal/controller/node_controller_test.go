package controller

import (
	"fmt"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Node controller tests", func() {
	Context("reconcilling node resources", func() {
		It("check node finalizers exist", func() {
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
			}, "30s", "5s").ShouldNot(HaveOccurred())
		})
	})
})
