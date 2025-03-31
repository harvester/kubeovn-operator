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

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	kubeovnv1 "github.com/harvester/kubeovn-operator/api/v1"
)

// nolint:unused
// log is for logging in this package.
var (
	configurationlog              = logf.Log.WithName("configuration-resource")
	ovnCentralDefaultResourceSpec = kubeovnv1.ResourceSpec{
		Requests: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("300m"),
			Memory: resource.MustParse("200Mi"),
		},
		Limits: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("3"),
			Memory: resource.MustParse("4Gi"),
		},
	}
	ovsOVNDefaultResourceSpec = kubeovnv1.ResourceSpec{
		Requests: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("200m"),
			Memory: resource.MustParse("200Mi"),
		},
		Limits: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("2"),
			Memory: resource.MustParse("1000Mi"),
		},
	}
	kubeOVNControllerDefaultResourceSpec = kubeovnv1.ResourceSpec{
		Requests: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("200m"),
			Memory: resource.MustParse("200Mi"),
		},
		Limits: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("1000m"),
			Memory: resource.MustParse("1Gi"),
		},
	}
	kubeOVNCNIDefaultResourceSpec = kubeovnv1.ResourceSpec{
		Requests: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("100m"),
			Memory: resource.MustParse("100Mi"),
		},
		Limits: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("1000m"),
			Memory: resource.MustParse("1Gi"),
		},
	}
	kubeOVNPingerDefaultResourceSpec = kubeovnv1.ResourceSpec{
		Requests: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("100m"),
			Memory: resource.MustParse("100Mi"),
		},
		Limits: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("200m"),
			Memory: resource.MustParse("400Mi"),
		},
	}
	kubeOVNMonitorDefaultResourceSpec = kubeovnv1.ResourceSpec{
		Requests: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("200m"),
			Memory: resource.MustParse("200Mi"),
		},
		Limits: kubeovnv1.CPUMemSpec{
			CPU:    resource.MustParse("200m"),
			Memory: resource.MustParse("200Mi"),
		},
	}
)

// SetupConfigurationWebhookWithManager registers the webhook for Configuration in the manager.
func SetupConfigurationWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&kubeovnv1.Configuration{}).
		WithDefaulter(&ConfigurationCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-kubeovn-io-v1-configuration,mutating=true,failurePolicy=fail,sideEffects=None,groups=kubeovn.io,resources=configurations,verbs=create;update,versions=v1,name=mconfiguration-v1.kb.io,admissionReviewVersions=v1

// ConfigurationCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Configuration when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type ConfigurationCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &ConfigurationCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Configuration.
func (d *ConfigurationCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	configuration, ok := obj.(*kubeovnv1.Configuration)

	if !ok {
		return fmt.Errorf("expected an Configuration object but got %T", obj)
	}
	configurationlog.Info("Defaulting for Configuration", "name", configuration.GetName())

	d.ApplyConfigurationDefaults(configuration)
	return nil
}

func (d *ConfigurationCustomDefaulter) ApplyConfigurationDefaults(config *kubeovnv1.Configuration) {
	config.Spec.OVNCentral = applyDefaults(config.Spec.OVNCentral, ovnCentralDefaultResourceSpec)
	config.Spec.OVSOVN = applyDefaults(config.Spec.OVSOVN, ovsOVNDefaultResourceSpec)
	config.Spec.KubeOVNController = applyDefaults(config.Spec.KubeOVNController, kubeOVNControllerDefaultResourceSpec)
	config.Spec.KubeOVNCNI = applyDefaults(config.Spec.KubeOVNCNI, kubeOVNCNIDefaultResourceSpec)
	config.Spec.KubeOVNPinger = applyDefaults(config.Spec.KubeOVNPinger, kubeOVNPingerDefaultResourceSpec)
	config.Spec.KubeOVNMonitor = applyDefaults(config.Spec.KubeOVNMonitor, kubeOVNMonitorDefaultResourceSpec)
}

// applyDefaults will apply baseline defaults for resource specs to configuration
func applyDefaults(resource, defaultValues kubeovnv1.ResourceSpec) kubeovnv1.ResourceSpec {
	if resource.Requests.CPU.IsZero() {
		resource.Requests.CPU = defaultValues.Requests.CPU
	}

	if resource.Requests.Memory.IsZero() {
		resource.Requests.Memory = defaultValues.Requests.Memory
	}

	if resource.Limits.CPU.IsZero() {
		resource.Limits.CPU = defaultValues.Limits.CPU
	}

	if resource.Limits.Memory.IsZero() {
		resource.Limits.Memory = defaultValues.Limits.Memory
	}
	return resource
}
