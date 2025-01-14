/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package cloudcmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/edgelesssys/constellation/v2/internal/attestation/measurements"
	"github.com/edgelesssys/constellation/v2/internal/compatibility"
	"github.com/edgelesssys/constellation/v2/internal/config"
	"github.com/edgelesssys/constellation/v2/internal/constants"
	"github.com/edgelesssys/constellation/v2/internal/logger"
	"github.com/edgelesssys/constellation/v2/internal/versions"
	"github.com/edgelesssys/constellation/v2/internal/versions/components"
	updatev1alpha1 "github.com/edgelesssys/constellation/v2/operators/constellation-node-operator/v2/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestUpgradeNodeVersion(t *testing.T) {
	someErr := errors.New("some error")
	testCases := map[string]struct {
		stable                *stubStableClient
		conditions            []metav1.Condition
		currentImageVersion   string
		currentClusterVersion string
		conf                  *config.Config
		getErr                error
		wantErr               bool
		wantUpdate            bool
		assertCorrectError    func(t *testing.T, err error) bool
	}{
		"success": {
			conf: func() *config.Config {
				conf := config.Default()
				conf.Image = "v1.2.3"
				conf.KubernetesVersion = versions.SupportedK8sVersions()[1]
				return conf
			}(),
			currentImageVersion:   "v1.2.2",
			currentClusterVersion: versions.SupportedK8sVersions()[0],
			stable: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":false}}`,
					},
				},
			},
			wantUpdate: true,
		},
		"only k8s upgrade": {
			conf: func() *config.Config {
				conf := config.Default()
				conf.Image = "v1.2.2"
				conf.KubernetesVersion = versions.SupportedK8sVersions()[1]
				return conf
			}(),
			currentImageVersion:   "v1.2.2",
			currentClusterVersion: versions.SupportedK8sVersions()[0],
			stable: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":false}}`,
					},
				},
			},
			wantUpdate: true,
			wantErr:    true,
			assertCorrectError: func(t *testing.T, err error) bool {
				upgradeErr := &compatibility.InvalidUpgradeError{}
				return assert.ErrorAs(t, err, &upgradeErr)
			},
		},
		"only image upgrade": {
			conf: func() *config.Config {
				conf := config.Default()
				conf.Image = "v1.2.3"
				conf.KubernetesVersion = versions.SupportedK8sVersions()[0]
				return conf
			}(),
			currentImageVersion:   "v1.2.2",
			currentClusterVersion: versions.SupportedK8sVersions()[0],
			stable: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":false}}`,
					},
				},
			},
			wantUpdate: true,
			wantErr:    true,
			assertCorrectError: func(t *testing.T, err error) bool {
				upgradeErr := &compatibility.InvalidUpgradeError{}
				return assert.ErrorAs(t, err, &upgradeErr)
			},
		},
		"not an upgrade": {
			conf: func() *config.Config {
				conf := config.Default()
				conf.Image = "v1.2.2"
				conf.KubernetesVersion = versions.SupportedK8sVersions()[0]
				return conf
			}(),
			currentImageVersion:   "v1.2.2",
			currentClusterVersion: versions.SupportedK8sVersions()[0],
			stable:                &stubStableClient{},
			wantErr:               true,
			assertCorrectError: func(t *testing.T, err error) bool {
				upgradeErr := &compatibility.InvalidUpgradeError{}
				return assert.ErrorAs(t, err, &upgradeErr)
			},
		},
		"upgrade in progress": {
			conf: func() *config.Config {
				conf := config.Default()
				conf.Image = "v1.2.3"
				conf.KubernetesVersion = versions.SupportedK8sVersions()[1]
				return conf
			}(),
			conditions: []metav1.Condition{{
				Type:   updatev1alpha1.ConditionOutdated,
				Status: metav1.ConditionTrue,
			}},
			currentImageVersion:   "v1.2.2",
			currentClusterVersion: versions.SupportedK8sVersions()[0],
			stable:                &stubStableClient{},
			wantErr:               true,
			assertCorrectError: func(t *testing.T, err error) bool {
				return assert.ErrorIs(t, err, ErrInProgress)
			},
		},
		"get error": {
			conf: func() *config.Config {
				conf := config.Default()
				conf.Image = "v1.2.3"
				return conf
			}(),
			getErr:  someErr,
			wantErr: true,
			assertCorrectError: func(t *testing.T, err error) bool {
				return assert.ErrorIs(t, err, someErr)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			nodeVersion := updatev1alpha1.NodeVersion{
				Spec: updatev1alpha1.NodeVersionSpec{
					ImageVersion:             tc.currentImageVersion,
					KubernetesClusterVersion: tc.currentClusterVersion,
				},
				Status: updatev1alpha1.NodeVersionStatus{
					Conditions: tc.conditions,
				},
			}

			unstrNodeVersion, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&nodeVersion)
			require.NoError(err)

			dynamicClient := &stubDynamicClient{object: &unstructured.Unstructured{Object: unstrNodeVersion}, getErr: tc.getErr}
			upgrader := Upgrader{
				stableInterface:  tc.stable,
				dynamicInterface: dynamicClient,
				imageFetcher:     &stubImageFetcher{},
				log:              logger.NewTest(t),
				outWriter:        io.Discard,
			}

			err = upgrader.UpgradeNodeVersion(context.Background(), tc.conf)

			// Check upgrades first because if we checked err first, UpgradeImage may error due to other reasons and still trigger an upgrade.
			if tc.wantUpdate {
				assert.NotNil(dynamicClient.updatedObject)
			} else {
				assert.Nil(dynamicClient.updatedObject)
			}

			if tc.wantErr {
				assert.Error(err)
				tc.assertCorrectError(t, err)
				return
			}
			assert.NoError(err)
		})
	}
}

func TestUpdateMeasurements(t *testing.T) {
	someErr := errors.New("error")
	testCases := map[string]struct {
		updater         *stubStableClient
		newMeasurements measurements.M
		wantUpdate      bool
		wantErr         bool
	}{
		"success": {
			updater: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":false}}`,
					},
				},
			},
			newMeasurements: measurements.M{
				0: measurements.WithAllBytes(0xBB, false),
			},
			wantUpdate: true,
		},
		"measurements are the same": {
			updater: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":false}}`,
					},
				},
			},
			newMeasurements: measurements.M{
				0: measurements.WithAllBytes(0xAA, false),
			},
		},
		"trying to set warnOnly to true results in error": {
			updater: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":false}}`,
					},
				},
			},
			newMeasurements: measurements.M{
				0: measurements.WithAllBytes(0xAA, true),
			},
			wantErr: true,
		},
		"setting warnOnly to false is allowed": {
			updater: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":true}}`,
					},
				},
			},
			newMeasurements: measurements.M{
				0: measurements.WithAllBytes(0xAA, false),
			},
			wantUpdate: true,
		},
		"getCurrent error": {
			updater: &stubStableClient{getErr: someErr},
			wantErr: true,
		},
		"update error": {
			updater: &stubStableClient{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						constants.MeasurementsFilename: `{"0":{"expected":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","warnOnly":false}}`,
					},
				},
				updateErr: someErr,
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			upgrader := &Upgrader{
				stableInterface: tc.updater,
				outWriter:       io.Discard,
				log:             logger.NewTest(t),
			}

			err := upgrader.updateMeasurements(context.Background(), tc.newMeasurements)
			if tc.wantErr {
				assert.Error(err)
				return
			}

			assert.NoError(err)
			if tc.wantUpdate {
				newMeasurementsJSON, err := json.Marshal(tc.newMeasurements)
				require.NoError(t, err)
				assert.JSONEq(string(newMeasurementsJSON), tc.updater.updatedConfigMap.Data[constants.MeasurementsFilename])
			} else {
				assert.Nil(tc.updater.updatedConfigMap)
			}
		})
	}
}

func TestUpdateImage(t *testing.T) {
	someErr := errors.New("error")
	testCases := map[string]struct {
		newImageReference string
		newImageVersion   string
		oldImageReference string
		oldImageVersion   string
		updateErr         error
		wantUpdate        bool
		wantErr           bool
	}{
		"success": {
			oldImageReference: "old-image-ref",
			oldImageVersion:   "v0.0.0",
			newImageReference: "new-image-ref",
			newImageVersion:   "v0.1.0",
			wantUpdate:        true,
		},
		"same version fails": {
			oldImageVersion: "v0.0.0",
			newImageVersion: "v0.0.0",
			wantErr:         true,
		},
		"update error": {
			updateErr: someErr,
			wantErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			upgrader := &Upgrader{
				log: logger.NewTest(t),
			}

			nodeVersion := updatev1alpha1.NodeVersion{
				Spec: updatev1alpha1.NodeVersionSpec{
					ImageReference: tc.oldImageReference,
					ImageVersion:   tc.oldImageVersion,
				},
			}

			err := upgrader.updateImage(&nodeVersion, tc.newImageReference, tc.newImageVersion)

			if tc.wantErr {
				assert.Error(err)
				return
			}

			assert.NoError(err)
			if tc.wantUpdate {
				assert.Equal(tc.newImageReference, nodeVersion.Spec.ImageReference)
				assert.Equal(tc.newImageVersion, nodeVersion.Spec.ImageVersion)
			} else {
				assert.Equal(tc.oldImageReference, nodeVersion.Spec.ImageReference)
				assert.Equal(tc.oldImageVersion, nodeVersion.Spec.ImageVersion)
			}
		})
	}
}

func TestUpdateK8s(t *testing.T) {
	someErr := errors.New("error")
	testCases := map[string]struct {
		newClusterVersion string
		oldClusterVersion string
		updateErr         error
		wantUpdate        bool
		wantErr           bool
	}{
		"success": {
			oldClusterVersion: "v0.0.0",
			newClusterVersion: "v0.1.0",
			wantUpdate:        true,
		},
		"same version fails": {
			oldClusterVersion: "v0.0.0",
			newClusterVersion: "v0.0.0",
			wantErr:           true,
		},
		"update error": {
			updateErr: someErr,
			wantErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			upgrader := &Upgrader{
				log: logger.NewTest(t),
			}

			nodeVersion := updatev1alpha1.NodeVersion{
				Spec: updatev1alpha1.NodeVersionSpec{
					KubernetesClusterVersion: tc.oldClusterVersion,
				},
			}

			_, err := upgrader.updateK8s(&nodeVersion, tc.newClusterVersion, components.Components{})

			if tc.wantErr {
				assert.Error(err)
				return
			}

			assert.NoError(err)
			if tc.wantUpdate {
				assert.Equal(tc.newClusterVersion, nodeVersion.Spec.KubernetesClusterVersion)
			} else {
				assert.Equal(tc.oldClusterVersion, nodeVersion.Spec.KubernetesClusterVersion)
			}
		})
	}
}

type stubDynamicClient struct {
	object        *unstructured.Unstructured
	updatedObject *unstructured.Unstructured
	getErr        error
	updateErr     error
}

func (u *stubDynamicClient) getCurrent(_ context.Context, _ string) (*unstructured.Unstructured, error) {
	return u.object, u.getErr
}

func (u *stubDynamicClient) update(_ context.Context, updatedObject *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	u.updatedObject = updatedObject
	return u.updatedObject, u.updateErr
}

type stubStableClient struct {
	configMap        *corev1.ConfigMap
	updatedConfigMap *corev1.ConfigMap
	k8sVersion       string
	getErr           error
	updateErr        error
	createErr        error
	k8sErr           error
}

func (s *stubStableClient) getCurrentConfigMap(_ context.Context, _ string) (*corev1.ConfigMap, error) {
	return s.configMap, s.getErr
}

func (s *stubStableClient) updateConfigMap(_ context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	s.updatedConfigMap = configMap
	return s.updatedConfigMap, s.updateErr
}

func (s *stubStableClient) createConfigMap(_ context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	s.configMap = configMap
	return s.configMap, s.createErr
}

func (s *stubStableClient) kubernetesVersion() (string, error) {
	return s.k8sVersion, s.k8sErr
}
