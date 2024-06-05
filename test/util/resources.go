// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	TestNamespaceName     = "test-namespace"
	CronJobNamePrefix     = "cronjob"
	DaemonSetNamePrefix   = "daemonset"
	DeploymentNamePrefix  = "deployment"
	JobNamePrefix         = "job"
	ReplicaSetNamePrefix  = "replicaset"
	StatefulSetNamePrefix = "statefulset"
)

var (
	True                 = true
	False                = false
	ArbitraryNumer int64 = 1302

	instrumentationInitContainer = corev1.Container{
		Name:  "dash0-instrumentation",
		Image: "some-registry.com:1234/dash0-instrumentation:4.5.6",
		Env: []corev1.EnvVar{{
			Name:  "DASH0_INSTRUMENTATION_FOLDER_DESTINATION",
			Value: "/opt/dash0",
		}},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &False,
			Privileged:               &False,
			ReadOnlyRootFilesystem:   &True,
			RunAsNonRoot:             &False,
			RunAsUser:                &ArbitraryNumer,
			RunAsGroup:               &ArbitraryNumer,
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "dash0-instrumentation",
			ReadOnly:  false,
			MountPath: "/opt/dash0",
		}},
	}
)

func EnsureDash0CustomResourceExists(
	ctx context.Context,
	k8sClient client.Client,
	qualifiedName types.NamespacedName,
	receiver client.Object,
	blueprint client.Object,
) client.Object {
	err := k8sClient.Get(ctx, qualifiedName, receiver)
	if err == nil {
		return receiver
	}
	if errors.IsNotFound(err) {
		Expect(k8sClient.Create(ctx, blueprint)).To(Succeed())
		return blueprint
	} else {
		Expect(err).ToNot(HaveOccurred())
		return nil
	}
}

func TestNamespace(name string) *corev1.Namespace {
	namespace := &corev1.Namespace{}
	namespace.Name = name
	return namespace
}

func CreateTestNamespace(ctx context.Context, k8sClient client.Client, name string) *corev1.Namespace {
	namespace := TestNamespace(name)
	Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())
	return namespace
}

func EnsureTestNamespaceExists(
	ctx context.Context,
	k8sClient client.Client,
	name string,
) *corev1.Namespace {
	object := EnsureDash0CustomResourceExists(
		ctx,
		k8sClient,
		types.NamespacedName{Name: name},
		&corev1.Namespace{},
		TestNamespace(name),
	)
	return object.(*corev1.Namespace)
}

func UniqueName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New())
}

func BasicCronJob(namespace string, name string) *batchv1.CronJob {
	workload := &batchv1.CronJob{}
	workload.Namespace = namespace
	workload.Name = name
	workload.Spec = batchv1.CronJobSpec{}
	workload.Spec.Schedule = "*/1 * * * *"
	workload.Spec.JobTemplate.Spec.Template = basicPodSpecTemplate()
	workload.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	return workload
}

func CreateBasicCronJob(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.CronJob {
	return CreateWorkload(ctx, k8sClient, BasicCronJob(namespace, name)).(*batchv1.CronJob)
}

func InstrumentedCronJob(namespace string, name string) *batchv1.CronJob {
	workload := BasicCronJob(namespace, name)
	simulateInstrumentedResource(&workload.Spec.JobTemplate.Spec.Template, &workload.ObjectMeta, namespace)
	return workload
}

func CreateInstrumentedCronJob(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.CronJob {
	return CreateWorkload(ctx, k8sClient, InstrumentedCronJob(namespace, name)).(*batchv1.CronJob)
}

func CronJobWithOptOutLabel(namespace string, name string) *batchv1.CronJob {
	workload := BasicCronJob(namespace, name)
	addOptOutLabel(&workload.ObjectMeta)
	return workload
}

func CreateCronJobWithOptOutLabel(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.CronJob {
	return CreateWorkload(ctx, k8sClient, CronJobWithOptOutLabel(namespace, name)).(*batchv1.CronJob)
}

func BasicDaemonSet(namespace string, name string) *appsv1.DaemonSet {
	workload := &appsv1.DaemonSet{}
	workload.Namespace = namespace
	workload.Name = name
	workload.Spec = appsv1.DaemonSetSpec{}
	workload.Spec.Template = basicPodSpecTemplate()
	workload.Spec.Selector = createSelector()
	return workload
}

func CreateBasicDaemonSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.DaemonSet {
	return CreateWorkload(ctx, k8sClient, BasicDaemonSet(namespace, name)).(*appsv1.DaemonSet)
}

func InstrumentedDaemonSet(namespace string, name string) *appsv1.DaemonSet {
	workload := BasicDaemonSet(namespace, name)
	simulateInstrumentedResource(&workload.Spec.Template, &workload.ObjectMeta, namespace)
	return workload
}

func CreateInstrumentedDaemonSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.DaemonSet {
	return CreateWorkload(ctx, k8sClient, InstrumentedDaemonSet(namespace, name)).(*appsv1.DaemonSet)
}

func DaemonSetWithOptOutLabel(namespace string, name string) *appsv1.DaemonSet {
	workload := BasicDaemonSet(namespace, name)
	addOptOutLabel(&workload.ObjectMeta)
	return workload
}

func CreateDaemonSetWithOptOutLabel(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.DaemonSet {
	return CreateWorkload(ctx, k8sClient, DaemonSetWithOptOutLabel(namespace, name)).(*appsv1.DaemonSet)
}

func BasicDeployment(namespace string, name string) *appsv1.Deployment {
	workload := &appsv1.Deployment{}
	workload.Namespace = namespace
	workload.Name = name
	workload.Spec = appsv1.DeploymentSpec{}
	workload.Spec.Template = basicPodSpecTemplate()
	workload.Spec.Selector = createSelector()
	return workload
}

func CreateBasicDeployment(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.Deployment {
	return CreateWorkload(ctx, k8sClient, BasicDeployment(namespace, name)).(*appsv1.Deployment)
}

func InstrumentedDeployment(namespace string, name string) *appsv1.Deployment {
	workload := BasicDeployment(namespace, name)
	simulateInstrumentedResource(&workload.Spec.Template, &workload.ObjectMeta, namespace)
	return workload
}

func CreateInstrumentedDeployment(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.Deployment {
	return CreateWorkload(ctx, k8sClient, InstrumentedDeployment(namespace, name)).(*appsv1.Deployment)
}

func DeploymentWithOptOutLabel(namespace string, name string) *appsv1.Deployment {
	workload := BasicDeployment(namespace, name)
	addOptOutLabel(&workload.ObjectMeta)
	return workload
}

func CreateDeploymentWithOptOutLabel(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.Deployment {
	return CreateWorkload(ctx, k8sClient, DeploymentWithOptOutLabel(namespace, name)).(*appsv1.Deployment)
}

func BasicJob(namespace string, name string) *batchv1.Job {
	workload := &batchv1.Job{}
	workload.Namespace = namespace
	workload.Name = name
	workload.Spec = batchv1.JobSpec{}
	workload.Spec.Template = basicPodSpecTemplate()
	workload.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	return workload
}

func CreateBasicJob(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.Job {
	return CreateWorkload(ctx, k8sClient, BasicJob(namespace, name)).(*batchv1.Job)
}

func InstrumentedJob(namespace string, name string) *batchv1.Job {
	workload := BasicJob(namespace, name)
	simulateInstrumentedResource(&workload.Spec.Template, &workload.ObjectMeta, namespace)
	return workload
}

func CreateInstrumentedJob(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.Job {
	return CreateWorkload(ctx, k8sClient, InstrumentedJob(namespace, name)).(*batchv1.Job)
}

func JobForWhichAnInstrumentationAttemptHasFailed(namespace string, name string) *batchv1.Job {
	workload := BasicJob(namespace, name)
	addInstrumentationLabels(&workload.ObjectMeta, false)
	return workload
}

func CreateJobForWhichAnInstrumentationAttemptHasFailed(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.Job {
	return CreateWorkload(ctx, k8sClient, JobForWhichAnInstrumentationAttemptHasFailed(namespace, name)).(*batchv1.Job)
}

func JobWithOptOutLabel(namespace string, name string) *batchv1.Job {
	workload := BasicJob(namespace, name)
	addOptOutLabel(&workload.ObjectMeta)
	return workload
}

func CreateJobWithOptOutLabel(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.Job {
	return CreateWorkload(ctx, k8sClient, JobWithOptOutLabel(namespace, name)).(*batchv1.Job)
}

func BasicReplicaSet(namespace string, name string) *appsv1.ReplicaSet {
	workload := &appsv1.ReplicaSet{}
	workload.Namespace = namespace
	workload.Name = name
	workload.Spec = appsv1.ReplicaSetSpec{}
	workload.Spec.Template = basicPodSpecTemplate()
	workload.Spec.Selector = createSelector()
	return workload
}

func CreateBasicReplicaSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.ReplicaSet {
	return CreateWorkload(ctx, k8sClient, BasicReplicaSet(namespace, name)).(*appsv1.ReplicaSet)
}

func InstrumentedReplicaSet(namespace string, name string) *appsv1.ReplicaSet {
	workload := BasicReplicaSet(namespace, name)
	simulateInstrumentedResource(&workload.Spec.Template, &workload.ObjectMeta, namespace)
	return workload
}

func CreateInstrumentedReplicaSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.ReplicaSet {
	return CreateWorkload(ctx, k8sClient, InstrumentedReplicaSet(namespace, name)).(*appsv1.ReplicaSet)
}

func ReplicaSetOwnedByDeployment(namespace string, name string) *appsv1.ReplicaSet {
	workload := BasicReplicaSet(namespace, name)
	workload.ObjectMeta = metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "deployment",
			UID:        "1234",
		}},
	}
	return workload
}

func CreateReplicaSetOwnedByDeployment(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.ReplicaSet {
	return CreateWorkload(ctx, k8sClient, ReplicaSetOwnedByDeployment(namespace, name)).(*appsv1.ReplicaSet)
}

func InstrumentedReplicaSetOwnedByDeployment(namespace string, name string) *appsv1.ReplicaSet {
	workload := ReplicaSetOwnedByDeployment(namespace, name)
	simulateInstrumentedResource(&workload.Spec.Template, &workload.ObjectMeta, namespace)
	return workload
}

func ReplicaSetWithOptOutLabel(namespace string, name string) *appsv1.ReplicaSet {
	workload := BasicReplicaSet(namespace, name)
	addOptOutLabel(&workload.ObjectMeta)
	return workload
}

func CreateReplicaSetWithOptOutLabel(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.ReplicaSet {
	return CreateWorkload(ctx, k8sClient, ReplicaSetWithOptOutLabel(namespace, name)).(*appsv1.ReplicaSet)
}

func BasicStatefulSet(namespace string, name string) *appsv1.StatefulSet {
	workload := &appsv1.StatefulSet{}
	workload.Namespace = namespace
	workload.Name = name
	workload.Spec = appsv1.StatefulSetSpec{}
	workload.Spec.Template = basicPodSpecTemplate()
	workload.Spec.Selector = createSelector()
	return workload
}

func CreateBasicStatefulSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.StatefulSet {
	return CreateWorkload(ctx, k8sClient, BasicStatefulSet(namespace, name)).(*appsv1.StatefulSet)
}

func InstrumentedStatefulSet(namespace string, name string) *appsv1.StatefulSet {
	workload := BasicStatefulSet(namespace, name)
	simulateInstrumentedResource(&workload.Spec.Template, &workload.ObjectMeta, namespace)
	return workload
}

func CreateInstrumentedStatefulSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.StatefulSet {
	return CreateWorkload(ctx, k8sClient, InstrumentedStatefulSet(namespace, name)).(*appsv1.StatefulSet)
}

func StatefulSetWithOptOutLabel(namespace string, name string) *appsv1.StatefulSet {
	workload := BasicStatefulSet(namespace, name)
	addOptOutLabel(&workload.ObjectMeta)
	return workload
}

func CreateStatefulSetWithOptOutLabel(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.StatefulSet {
	return CreateWorkload(ctx, k8sClient, StatefulSetWithOptOutLabel(namespace, name)).(*appsv1.StatefulSet)
}

func basicPodSpecTemplate() corev1.PodTemplateSpec {
	podSpecTemplate := corev1.PodTemplateSpec{}
	podSpecTemplate.Labels = map[string]string{"app": "test"}
	podSpecTemplate.Spec.Containers = []corev1.Container{{
		Name:  "test-container-0",
		Image: "ubuntu",
	}}
	return podSpecTemplate
}

func createSelector() *metav1.LabelSelector {
	selector := &metav1.LabelSelector{}
	selector.MatchLabels = map[string]string{"app": "test"}
	return selector
}

func CreateWorkload(ctx context.Context, k8sClient client.Client, workload client.Object) client.Object {
	Expect(k8sClient.Create(ctx, workload)).Should(Succeed())
	return workload
}

func DeploymentWithMoreBellsAndWhistles(namespace string, name string) *appsv1.Deployment {
	workload := BasicDeployment(namespace, name)
	podSpec := &workload.Spec.Template.Spec
	podSpec.Volumes = []corev1.Volume{
		{
			Name:         "test-volume-0",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name:         "test-volume-1",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}
	podSpec.InitContainers = []corev1.Container{
		{
			Name:  "test-init-container-0",
			Image: "ubuntu",
			Env: []corev1.EnvVar{{
				Name:  "TEST_INIT_0",
				Value: "value",
			}},
		},
		{
			Name:  "test-init-container-1",
			Image: "ubuntu",
			Env: []corev1.EnvVar{
				{
					Name:  "TEST_INIT_0",
					Value: "value",
				},
				{
					Name:  "TEST_INIT_1",
					Value: "value",
				},
			},
		},
	}
	podSpec.Containers = []corev1.Container{
		{
			Name:  "test-container-0",
			Image: "ubuntu",
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "test-volume-0",
				MountPath: "/test-1",
			}},
			Env: []corev1.EnvVar{{
				Name:  "TEST0",
				Value: "value",
			}},
		},
		{
			Name:  "test-container-1",
			Image: "ubuntu",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "test-volume-0",
					MountPath: "/test-0",
				},
				{
					Name:      "test-volume-1",
					MountPath: "/test-1",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "TEST0",
					Value: "value",
				},
				{
					Name:      "TEST1",
					ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
				},
			},
		},
	}

	return workload
}

func DeploymentWithExistingDash0Artifacts(namespace string, name string) *appsv1.Deployment {
	deployment := BasicDeployment(namespace, name)
	podSpec := &deployment.Spec.Template.Spec
	podSpec.Volumes = []corev1.Volume{
		{
			Name:         "test-volume-0",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name: "dash0-instrumentation",
			// the volume source should be updated/overwritten by the webhook
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/foo/bar"}},
		},
		{
			Name:         "test-volume-2",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}
	podSpec.InitContainers = []corev1.Container{
		{
			Name:  "test-init-container-0",
			Image: "ubuntu",
			Env: []corev1.EnvVar{{
				Name:  "TEST_INIT_0",
				Value: "value",
			}},
		},
		instrumentationInitContainer,
		{
			Name:  "test-init-container-2",
			Image: "ubuntu",
			Env: []corev1.EnvVar{
				{
					Name:  "TEST_INIT_0",
					Value: "value",
				},
				{
					Name:  "TEST_INIT_1",
					Value: "value",
				},
				{
					Name:  "TEST_INIT_2",
					Value: "value",
				},
			},
		},
	}
	podSpec.Containers = []corev1.Container{
		{
			Name:  "test-container-0",
			Image: "ubuntu",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "test-volume-0",
					MountPath: "/test-1",
				},
				{
					Name:      "dash0-instrumentation",
					MountPath: "/will/be/overwritten",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "TEST0",
					Value: "value",
				},
				{
					// Dash0 does not support injecting into containers that already have NODE_OPTIONS set via a
					// ValueFrom clause, thus this env var will not be modified.
					Name:      "NODE_OPTIONS",
					ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
				},
				{
					// this ValueFrom will be removed and replaced by a simple Value
					Name:      "DASH0_OTEL_COLLECTOR_BASE_URL",
					ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
				},
			},
		},
		{
			Name:  "test-container-1",
			Image: "ubuntu",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "test-volume-0",
					MountPath: "/test-0",
				},
				{
					Name:      "dash0-instrumentation",
					MountPath: "/will/be/overwritten",
				},
				{
					Name:      "test-volume-2",
					MountPath: "/test-2",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "DASH0_OTEL_COLLECTOR_BASE_URL",
					Value: "base url will be replaced",
				},
				{
					Name:  "NODE_OPTIONS",
					Value: "--require something-else --experimental-modules",
				},
				{
					Name:  "TEST2",
					Value: "value",
				},
			},
		},
	}

	return deployment
}

func InstrumentedDeploymentWithMoreBellsAndWhistles(namespace string, name string) *appsv1.Deployment {
	deployment := DeploymentWithMoreBellsAndWhistles(namespace, name)
	podSpec := &deployment.Spec.Template.Spec
	podSpec.Volumes = []corev1.Volume{
		{
			Name:         "test-volume-0",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name:         "test-volume-1",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name: "dash0-instrumentation",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: resource.NewScaledQuantity(150, resource.Mega),
				},
			},
		},
	}
	podSpec.InitContainers = []corev1.Container{
		{
			Name:  "test-init-container-0",
			Image: "ubuntu",
			Env: []corev1.EnvVar{{
				Name:  "TEST_INIT_0",
				Value: "value",
			}},
		},
		{
			Name:  "test-init-container-1",
			Image: "ubuntu",
			Env: []corev1.EnvVar{
				{
					Name:  "TEST_INIT_0",
					Value: "value",
				},
				{
					Name:  "TEST_INIT_1",
					Value: "value",
				},
			},
		},
		instrumentationInitContainer,
	}
	podSpec.Containers = []corev1.Container{
		{
			Name:  "test-container-0",
			Image: "ubuntu",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "test-volume-0",
					MountPath: "/test-1",
				},
				{
					Name:      "dash0-instrumentation",
					MountPath: "/opt/dash0",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "TEST0",
					Value: "value",
				},
				{
					Name:  "NODE_OPTIONS",
					Value: "--require /opt/dash0/instrumentation/node.js/node_modules/@dash0/opentelemetry/src/index.js",
				},
				{
					Name:  "DASH0_OTEL_COLLECTOR_BASE_URL",
					Value: fmt.Sprintf("http://dash0-opentelemetry-collector-daemonset.%s.svc.cluster.local:4318", namespace),
				},
			},
		},
		{
			Name:  "test-container-1",
			Image: "ubuntu",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "test-volume-0",
					MountPath: "/test-0",
				},
				{
					Name:      "test-volume-1",
					MountPath: "/test-1",
				},
				{
					Name:      "dash0-instrumentation",
					MountPath: "/opt/dash0",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "TEST0",
					Value: "value",
				},
				{
					Name:      "TEST1",
					ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
				},
				{
					Name:  "NODE_OPTIONS",
					Value: "--require /opt/dash0/instrumentation/node.js/node_modules/@dash0/opentelemetry/src/index.js",
				},
				{
					Name:  "DASH0_OTEL_COLLECTOR_BASE_URL",
					Value: fmt.Sprintf("http://dash0-opentelemetry-collector-daemonset.%s.svc.cluster.local:4318", namespace),
				},
			},
		},
	}

	return deployment
}

func simulateInstrumentedResource(podTemplateSpec *corev1.PodTemplateSpec, meta *metav1.ObjectMeta, namespace string) {
	podSpec := &podTemplateSpec.Spec
	podSpec.Volumes = []corev1.Volume{
		{
			Name: "dash0-instrumentation",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: resource.NewScaledQuantity(150, resource.Mega),
				},
			},
		},
	}
	podSpec.InitContainers = []corev1.Container{instrumentationInitContainer}

	container := &podSpec.Containers[0]
	container.VolumeMounts = []corev1.VolumeMount{{
		Name:      "dash0-instrumentation",
		MountPath: "/opt/dash0",
	}}
	container.Env = []corev1.EnvVar{
		{
			Name:  "NODE_OPTIONS",
			Value: "--require /opt/dash0/instrumentation/node.js/node_modules/@dash0/opentelemetry/src/index.js",
		},
		{
			Name:  "DASH0_OTEL_COLLECTOR_BASE_URL",
			Value: fmt.Sprintf("http://dash0-opentelemetry-collector-daemonset.%s.svc.cluster.local:4318", namespace),
		},
	}

	addInstrumentationLabels(meta, true)
	addInstrumentationLabels(&podTemplateSpec.ObjectMeta, true)
}

func GetCronJob(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.CronJob {
	workload := &batchv1.CronJob{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	ExpectWithOffset(1, k8sClient.Get(ctx, namespacedName, workload)).Should(Succeed())
	return workload
}

func GetDaemonSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.DaemonSet {
	workload := &appsv1.DaemonSet{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	ExpectWithOffset(1, k8sClient.Get(ctx, namespacedName, workload)).Should(Succeed())
	return workload
}

func GetDeployment(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.Deployment {
	workload := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	ExpectWithOffset(1, k8sClient.Get(ctx, namespacedName, workload)).Should(Succeed())
	return workload
}

func GetJob(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *batchv1.Job {
	workload := &batchv1.Job{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	ExpectWithOffset(1, k8sClient.Get(ctx, namespacedName, workload)).Should(Succeed())
	return workload
}

func GetReplicaSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.ReplicaSet {
	workload := &appsv1.ReplicaSet{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	ExpectWithOffset(1, k8sClient.Get(ctx, namespacedName, workload)).Should(Succeed())
	return workload
}

func GetStatefulSet(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) *appsv1.StatefulSet {
	workload := &appsv1.StatefulSet{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	ExpectWithOffset(1, k8sClient.Get(ctx, namespacedName, workload)).Should(Succeed())
	return workload
}

func addInstrumentationLabels(meta *metav1.ObjectMeta, successful bool) {
	AddLabel(meta, "dash0.com/instrumented", strconv.FormatBool(successful))
	AddLabel(meta, "dash0.com/operator-image", "some-registry.com_1234_dash0-operator-controller_1.2.3")
	AddLabel(meta, "dash0.com/init-container-image", "some-registry.com_1234_dash0-instrumentation_4.5.6")
	AddLabel(meta, "dash0.com/instrumented-by", "someone")
}

func addOptOutLabel(meta *metav1.ObjectMeta) {
	AddLabel(meta, "dash0.com/opt-out", "true")
}

func AddLabel(meta *metav1.ObjectMeta, key string, value string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string, 1)
	}
	meta.Labels[key] = value
}

func DeleteAllCreatedObjects(
	ctx context.Context,
	k8sClient client.Client,
	createdObjects []client.Object,
) []client.Object {
	By("Remove all created objects")
	for _, object := range createdObjects {
		Expect(k8sClient.Delete(ctx, object, &client.DeleteOptions{
			GracePeriodSeconds: new(int64),
		})).To(Succeed())
	}
	return make([]client.Object, 0)
}

func DeleteAllEvents(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	namespace string,
) {
	err := clientset.CoreV1().Events(namespace).DeleteCollection(ctx, metav1.DeleteOptions{
		GracePeriodSeconds: new(int64), // delete immediately
	}, metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	allEvents, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(allEvents.Items).To(BeEmpty())
}
