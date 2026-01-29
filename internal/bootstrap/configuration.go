package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
)

const (
	DefaultConfigMountPath        = "/etc/kubeovn-operator/default-config"
	ConfigurationFileName         = "configuration.yaml"
	DefaultConfigurationName      = "kubeovn"
	DefaultConfigurationNamespace = "kube-system"
)

// ConfigurationBootstrapper loads and applies the default Configuration from a mounted ConfigMap.
type ConfigurationBootstrapper struct {
	Client          client.Client
	ConfigMountPath string
	Log             logr.Logger
}

// Start implements manager.Runnable. It is called when the manager starts.
func (b *ConfigurationBootstrapper) Start(ctx context.Context) error {
	b.Log.Info("starting configuration bootstrapper")

	// Run the bootstrap logic with retry
	err := b.loadAndApplyDefaultConfiguration(ctx)
	if err != nil {
		b.Log.Error(err, "configuration bootstrapper encountered an error")
		// Return nil to not crash the manager; the error is logged
	}

	b.Log.Info("configuration bootstrapper completed")
	return nil
}

// NeedLeaderElection implements manager.LeaderElectionRunnable.
// Returns true so that bootstrap only runs on the leader to avoid race conditions.
func (b *ConfigurationBootstrapper) NeedLeaderElection() bool {
	return true
}

// loadAndApplyDefaultConfiguration reads the Configuration spec from the mounted ConfigMap
// and creates the Configuration CR if it does not already exist.
func (b *ConfigurationBootstrapper) loadAndApplyDefaultConfiguration(ctx context.Context) error {
	existingConfig := &kubeovniov1.Configuration{}
	err := b.Client.Get(ctx, types.NamespacedName{
		Name:      DefaultConfigurationName,
		Namespace: DefaultConfigurationNamespace,
	}, existingConfig)

	if err == nil {
		b.Log.Info("Configuration already exists, skipping default configuration loading",
			"configuration", DefaultConfigurationName,
			"namespace", DefaultConfigurationNamespace)
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("error checking for existing Configuration: %w", err)
	}

	configFilePath := filepath.Join(b.ConfigMountPath, ConfigurationFileName)
	b.Log.Info("Configuration not found, loading spec from mounted ConfigMap",
		"configuration", DefaultConfigurationName,
		"namespace", DefaultConfigurationNamespace,
		"path", configFilePath)

	specData, err := os.ReadFile(configFilePath)
	if err != nil {
		b.Log.Error(err, "failed to read configuration spec file",
			"path", configFilePath)
		return nil
	}
	spec := &kubeovniov1.ConfigurationSpec{}
	if err := yaml.Unmarshal(specData, spec); err != nil {
		b.Log.Error(err, "failed to parse Configuration spec YAML",
			"path", configFilePath)
		return nil
	}

	config := &kubeovniov1.Configuration{}
	config.Name = DefaultConfigurationName
	config.Namespace = DefaultConfigurationNamespace
	config.Spec = *spec

	b.Log.Info("creating default Configuration from mounted ConfigMap",
		"path", configFilePath,
		"configuration", config.Name,
		"namespace", config.Namespace)
	err = b.createConfigurationWithRetry(ctx, config)
	if err != nil {
		b.Log.Error(err, "failed to create default Configuration after retries",
			"configuration", config.Name)
		return nil
	}

	b.Log.Info("successfully created default Configuration",
		"configuration", config.Name,
		"namespace", config.Namespace)
	return nil
}

// createConfigurationWithRetry attempts to create the Configuration with exponential backoff.
// This handles the case where the webhook server is not ready immediately after manager start.
func (b *ConfigurationBootstrapper) createConfigurationWithRetry(ctx context.Context, config *kubeovniov1.Configuration) error {
	backoff := wait.Backoff{
		Steps:    6,
		Duration: 1 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
	}

	return wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (bool, error) {
		err := b.Client.Create(ctx, config)
		if err == nil {
			return true, nil
		}

		if apierrors.IsAlreadyExists(err) {
			b.Log.Info("Configuration already exists",
				"configuration", config.Name)
			return true, nil
		}

		b.Log.Info("retrying Configuration creation",
			"configuration", config.Name,
			"error", err.Error())
		return false, nil
	})
}
