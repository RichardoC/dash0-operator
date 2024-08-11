// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package otelcolresources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/dash0hq/dash0-operator/test/util"
)

const (
	namespace  = "some-namespace"
	namePrefix = "unit-test"
)

var _ = Describe("The desired state of the OpenTelemetry Collector resources", func() {
	It("should fail if no ingress endpoint has been provided", func() {
		_, err := assembleDesiredState(&oTelColConfig{
			Namespace:          namespace,
			NamePrefix:         namePrefix,
			AuthorizationToken: AuthorizationToken,
			SecretRef:          SecretRefEmpty,
			Images:             TestImages,
		})
		Expect(err).To(HaveOccurred())
	})

	It("should describe the desired state as a set of Kubernetes client objects", func() {
		desiredState, err := assembleDesiredState(&oTelColConfig{
			Namespace:          namespace,
			NamePrefix:         namePrefix,
			IngressEndpoint:    IngressEndpoint,
			AuthorizationToken: AuthorizationToken,
			Images:             TestImages,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(desiredState).To(HaveLen(6))
		configMapContent := getConfigMapContent(desiredState)
		Expect(configMapContent).To(ContainSubstring(fmt.Sprintf("endpoint: %s", IngressEndpoint)))
		Expect(configMapContent).NotTo(ContainSubstring("file/traces"))
		Expect(configMapContent).NotTo(ContainSubstring("file/metrics"))
		Expect(configMapContent).NotTo(ContainSubstring("file/logs"))

		daemonSet := getDaemonSet(desiredState)
		Expect(daemonSet).NotTo(BeNil())
		Expect(daemonSet.ObjectMeta.Labels["dash0.com/enable"]).To(Equal("false"))
		podSpec := daemonSet.Spec.Template.Spec

		Expect(podSpec.Volumes).To(HaveLen(2))
		configMapVolume := findVolumeByName(podSpec.Volumes, "opentelemetry-collector-configmap")
		Expect(configMapVolume).NotTo(BeNil())
		Expect(configMapVolume.VolumeSource.ConfigMap.LocalObjectReference.Name).
			To(Equal("unit-test-opentelemetry-collector-agent"))
		for _, container := range podSpec.Containers {
			Expect(findVolumeMountByName(container.VolumeMounts, "opentelemetry-collector-configmap")).NotTo(BeNil())
		}

		pidFileVolume := findVolumeByName(podSpec.Volumes, "opentelemetry-collector-pidfile")
		Expect(pidFileVolume).NotTo(BeNil())
		Expect(pidFileVolume.VolumeSource.EmptyDir).NotTo(BeNil())
		for _, container := range podSpec.Containers {
			Expect(findVolumeMountByName(container.VolumeMounts, "opentelemetry-collector-pidfile")).NotTo(BeNil())
		}

		Expect(podSpec.Containers).To(HaveLen(2))

		collectorContainer := podSpec.Containers[0]
		Expect(collectorContainer).NotTo(BeNil())
		Expect(collectorContainer.Image).To(Equal(CollectorImageTest))
		Expect(collectorContainer.ImagePullPolicy).To(Equal(corev1.PullAlways))
		collectorContainerArgs := collectorContainer.Args
		Expect(collectorContainerArgs).To(HaveLen(1))
		Expect(collectorContainerArgs[0]).To(Equal("--config=file:/etc/otelcol/conf/config.yaml"))
		Expect(collectorContainer.VolumeMounts).To(HaveLen(2))
		Expect(collectorContainer.VolumeMounts).To(
			ContainElement(MatchVolumeMount("opentelemetry-collector-configmap", "/etc/otelcol/conf")))
		Expect(collectorContainer.VolumeMounts).To(
			ContainElement(MatchVolumeMount("opentelemetry-collector-pidfile", "/etc/otelcol/run")))

		configReloaderContainer := podSpec.Containers[1]
		Expect(configReloaderContainer).NotTo(BeNil())
		Expect(configReloaderContainer.Image).To(Equal(ConfigurationReloaderImageTest))
		Expect(configReloaderContainer.ImagePullPolicy).To(Equal(corev1.PullAlways))
		configReloaderContainerArgs := configReloaderContainer.Args
		Expect(configReloaderContainerArgs).To(HaveLen(2))
		Expect(configReloaderContainerArgs[0]).To(Equal("--pidfile=/etc/otelcol/run/pid.file"))
		Expect(configReloaderContainerArgs[1]).To(Equal("/etc/otelcol/conf/config.yaml"))
		Expect(configReloaderContainer.VolumeMounts).To(HaveLen(2))
		Expect(configReloaderContainer.VolumeMounts).To(
			ContainElement(MatchVolumeMount("opentelemetry-collector-configmap", "/etc/otelcol/conf")))
		Expect(configReloaderContainer.VolumeMounts).To(
			ContainElement(MatchVolumeMount("opentelemetry-collector-pidfile", "/etc/otelcol/run")))
	})

	It("should use the authorization token directly if provided", func() {
		desiredState, err := assembleDesiredState(&oTelColConfig{
			Namespace:          namespace,
			NamePrefix:         namePrefix,
			IngressEndpoint:    IngressEndpoint,
			AuthorizationToken: AuthorizationToken,
		})

		Expect(err).ToNot(HaveOccurred())
		configMapContent := getConfigMapContent(desiredState)
		Expect(configMapContent).To(ContainSubstring("Authorization: Bearer ${env:AUTH_TOKEN}"))

		daemonSet := getDaemonSet(desiredState)

		authTokenEnvVar := findEnvVarByName(daemonSet.Spec.Template.Spec.Containers[0].Env, "AUTH_TOKEN")
		Expect(authTokenEnvVar).NotTo(BeNil())
		Expect(authTokenEnvVar.Value).To(Equal(AuthorizationToken))
	})

	It("should use the secret reference if provided (and no authorization token has been provided)", func() {
		desiredState, err := assembleDesiredState(&oTelColConfig{
			Namespace:       namespace,
			NamePrefix:      namePrefix,
			IngressEndpoint: IngressEndpoint,
			SecretRef:       "some-secret",
		})

		Expect(err).ToNot(HaveOccurred())
		configMapContent := getConfigMapContent(desiredState)
		Expect(configMapContent).To(ContainSubstring("Authorization: Bearer ${env:AUTH_TOKEN}"))

		daemonSet := getDaemonSet(desiredState)
		podSpec := daemonSet.Spec.Template.Spec
		container := podSpec.Containers[0]
		authTokenEnvVar := findEnvVarByName(container.Env, "AUTH_TOKEN")
		Expect(authTokenEnvVar).NotTo(BeNil())
		Expect(authTokenEnvVar.ValueFrom.SecretKeyRef.Name).To(Equal("some-secret"))
		Expect(authTokenEnvVar.ValueFrom.SecretKeyRef.Key).To(Equal("dash0-authorization-token"))
	})

	It("should not add the auth token env var if no authorization token has been provided", func() {
		desiredState, err := assembleDesiredState(&oTelColConfig{
			Namespace:       namespace,
			NamePrefix:      namePrefix,
			IngressEndpoint: IngressEndpoint,
		})

		Expect(err).ToNot(HaveOccurred())
		configMapContent := getConfigMapContent(desiredState)
		Expect(configMapContent).NotTo(ContainSubstring("Authorization: Bearer ${env:AUTH_TOKEN}"))

		daemonSet := getDaemonSet(desiredState)
		podSpec := daemonSet.Spec.Template.Spec
		container := podSpec.Containers[0]
		authTokenEnvVar := findEnvVarByName(container.Env, "AUTH_TOKEN")
		Expect(authTokenEnvVar).To(BeNil())
	})
})

func getConfigMap(desiredState []client.Object) *corev1.ConfigMap {
	for _, object := range desiredState {
		if cm, ok := object.(*corev1.ConfigMap); ok {
			return cm
		}
	}
	return nil
}

func getConfigMapContent(desiredState []client.Object) string {
	cm := getConfigMap(desiredState)
	return cm.Data["config.yaml"]
}

func getDaemonSet(desiredState []client.Object) *appsv1.DaemonSet {
	for _, object := range desiredState {
		if ds, ok := object.(*appsv1.DaemonSet); ok {
			return ds
		}
	}
	return nil
}

func findEnvVarByName(objects []corev1.EnvVar, name string) *corev1.EnvVar {
	for _, object := range objects {
		if object.Name == name {
			return &object
		}
	}
	return nil
}

func findVolumeByName(objects []corev1.Volume, name string) *corev1.Volume {
	for _, object := range objects {
		if object.Name == name {
			return &object
		}
	}
	return nil
}

func findVolumeMountByName(objects []corev1.VolumeMount, name string) *corev1.VolumeMount {
	for _, object := range objects {
		if object.Name == name {
			return &object
		}
	}
	return nil
}