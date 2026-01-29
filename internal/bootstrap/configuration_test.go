package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = kubeovniov1.AddToScheme(scheme)
	return scheme
}

func newTestLogger() logr.Logger {
	return zap.New(zap.UseDevMode(true))
}

func TestNeedLeaderElection(t *testing.T) {
	bootstrapper := &ConfigurationBootstrapper{}
	assert.True(t, bootstrapper.NeedLeaderElection(), "NeedLeaderElection should return true")
}

func TestLoadAndApplyDefaultConfiguration(t *testing.T) {
	const existingSpecLabel = "original-label"
	const validSpecYAML = "masterNodesLabel: node-role.kubernetes.io/control-plane=true"

	tests := []struct {
		name                string
		existingConfig      bool
		configFileContent   *string // nil = no file created, pointer to string = file content
		expectConfigCreated bool
		validateSpec        func(t *testing.T, spec kubeovniov1.ConfigurationSpec)
	}{
		{
			name:                "configuration already exists without configmap",
			existingConfig:      true,
			configFileContent:   nil,
			expectConfigCreated: true, // already exists, not created by bootstrapper
			validateSpec: func(t *testing.T, spec kubeovniov1.ConfigurationSpec) {
				// Verify the original config was not modified
				assert.Equal(t, existingSpecLabel, spec.MasterNodesLabel)
			},
		},
		{
			name:                "configuration already exists with configmap",
			existingConfig:      true,
			configFileContent:   ptr.To("masterNodesLabel: new-label-from-configmap"),
			expectConfigCreated: true, // already exists, not created by bootstrapper
			validateSpec: func(t *testing.T, spec kubeovniov1.ConfigurationSpec) {
				// Verify the original config was not overwritten by configmap content
				assert.Equal(t, existingSpecLabel, spec.MasterNodesLabel)
			},
		},
		{
			name:                "config file not found",
			existingConfig:      false,
			configFileContent:   nil,
			expectConfigCreated: false,
		},
		{
			name:                "invalid YAML",
			existingConfig:      false,
			configFileContent:   ptr.To("not a valid yaml: [[["),
			expectConfigCreated: false,
		},
		{
			name:                "valid config",
			existingConfig:      false,
			configFileContent:   ptr.To(validSpecYAML),
			expectConfigCreated: true,
			validateSpec: func(t *testing.T, spec kubeovniov1.ConfigurationSpec) {
				assert.Equal(t, "node-role.kubernetes.io/control-plane=true", spec.MasterNodesLabel)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme()
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			// Setup existing Configuration if needed
			if tt.existingConfig {
				existingConfig := &kubeovniov1.Configuration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      DefaultConfigurationName,
						Namespace: DefaultConfigurationNamespace,
					},
					Spec: kubeovniov1.ConfigurationSpec{
						MasterNodesLabel: existingSpecLabel,
					},
				}
				clientBuilder = clientBuilder.WithObjects(existingConfig)
			}

			fakeClient := clientBuilder.Build()

			// Setup config file if needed
			configMountPath := t.TempDir()
			if tt.configFileContent != nil {
				err := os.WriteFile(filepath.Join(configMountPath, ConfigurationFileName), []byte(*tt.configFileContent), 0644)
				require.NoError(t, err)
			}
			// If configFileContent is nil, the directory exists but the file doesn't

			bootstrapper := &ConfigurationBootstrapper{
				Client:          fakeClient,
				ConfigMountPath: configMountPath,
				Log:             newTestLogger(),
			}

			// Execute
			ctx := context.Background()
			err := bootstrapper.loadAndApplyDefaultConfiguration(ctx)
			require.NoError(t, err)

			// Verify
			config := &kubeovniov1.Configuration{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      DefaultConfigurationName,
				Namespace: DefaultConfigurationNamespace,
			}, config)

			if tt.expectConfigCreated {
				require.NoError(t, err)
				if tt.validateSpec != nil {
					tt.validateSpec(t, config.Spec)
				}
			} else {
				assert.Error(t, err, "Configuration should not exist")
			}
		})
	}
}

func TestCreateConfigurationWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		existingConfig bool
		failCount      int
		expectError    bool
	}{
		{
			name:           "already exists",
			existingConfig: true,
			failCount:      0,
			expectError:    false,
		},
		{
			name:           "retries on failure then succeeds",
			existingConfig: false,
			failCount:      2,
			expectError:    false,
		},
		{
			name:           "fails after all retries exhausted",
			existingConfig: false,
			failCount:      10, // More than the 6 retry steps
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme()
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			if tt.existingConfig {
				existingConfig := &kubeovniov1.Configuration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      DefaultConfigurationName,
						Namespace: DefaultConfigurationNamespace,
					},
				}
				clientBuilder = clientBuilder.WithObjects(existingConfig)
			}

			fakeClient := clientBuilder.Build()

			var testClient client.Client
			if tt.failCount > 0 {
				testClient = &mockFailingClient{
					Client:    fakeClient,
					failCount: tt.failCount,
				}
			} else {
				testClient = fakeClient
			}

			bootstrapper := &ConfigurationBootstrapper{
				Client:          testClient,
				ConfigMountPath: "/tmp",
				Log:             newTestLogger(),
			}

			config := &kubeovniov1.Configuration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DefaultConfigurationName,
					Namespace: DefaultConfigurationNamespace,
				},
			}

			ctx := context.Background()
			err := bootstrapper.createConfigurationWithRetry(ctx, config)

			require.Equal(t, tt.expectError, err != nil)
		})
	}
}

// mockFailingClient wraps a client and fails the first N Create calls
type mockFailingClient struct {
	client.Client
	failCount    int
	currentCount int
}

func (m *mockFailingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if m.currentCount < m.failCount {
		m.currentCount++
		return &mockError{message: "simulated transient error"}
	}
	return m.Client.Create(ctx, obj, opts...)
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
