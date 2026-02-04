package bootstrap

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
)

const testNamespace = "test-namespace"

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = kubeovniov1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
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
		configMapContent    *string // nil = no ConfigMap created, pointer to string = ConfigMap data content
		expectConfigPresent bool
		validateSpec        func(t *testing.T, spec kubeovniov1.ConfigurationSpec)
	}{
		{
			name:                "configuration already exists without configmap",
			existingConfig:      true,
			configMapContent:    nil,
			expectConfigPresent: true, // already exists, not created by bootstrapper
			validateSpec: func(t *testing.T, spec kubeovniov1.ConfigurationSpec) {
				// Verify the original config was not modified
				assert.Equal(t, existingSpecLabel, spec.MasterNodesLabel)
			},
		},
		{
			name:                "configuration already exists with configmap",
			existingConfig:      true,
			configMapContent:    ptr.To("masterNodesLabel: new-label-from-configmap"),
			expectConfigPresent: true, // already exists, not created by bootstrapper
			validateSpec: func(t *testing.T, spec kubeovniov1.ConfigurationSpec) {
				// Verify the original config was not overwritten by configmap content
				assert.Equal(t, existingSpecLabel, spec.MasterNodesLabel)
			},
		},
		{
			name:                "configmap not found",
			existingConfig:      false,
			configMapContent:    nil,
			expectConfigPresent: false,
		},
		{
			name:                "invalid YAML in configmap",
			existingConfig:      false,
			configMapContent:    ptr.To("not a valid yaml: [[["),
			expectConfigPresent: false,
		},
		{
			name:                "valid config from configmap",
			existingConfig:      false,
			configMapContent:    ptr.To(validSpecYAML),
			expectConfigPresent: true,
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
						Namespace: testNamespace,
					},
					Spec: kubeovniov1.ConfigurationSpec{
						MasterNodesLabel: existingSpecLabel,
					},
				}
				clientBuilder = clientBuilder.WithObjects(existingConfig)
			}

			// Setup ConfigMap if needed
			if tt.configMapContent != nil {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      BootstrapConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						ConfigMapDataKey: *tt.configMapContent,
					},
				}
				clientBuilder = clientBuilder.WithObjects(configMap)
			}

			fakeClient := clientBuilder.Build()

			bootstrapper := &ConfigurationBootstrapper{
				Client:    fakeClient,
				Namespace: testNamespace,
				Log:       newTestLogger(),
			}

			// Execute
			ctx := context.Background()
			err := bootstrapper.loadAndApplyDefaultConfiguration(ctx)
			require.NoError(t, err)

			// Verify
			config := &kubeovniov1.Configuration{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      DefaultConfigurationName,
				Namespace: testNamespace,
			}, config)

			if tt.expectConfigPresent {
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
						Namespace: testNamespace,
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
				Client:    testClient,
				Namespace: testNamespace,
				Log:       newTestLogger(),
			}

			config := &kubeovniov1.Configuration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DefaultConfigurationName,
					Namespace: testNamespace,
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
