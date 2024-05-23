// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/dash0hq/dash0-operator/test/util"
)

// Maintenance note: There is some overlap of test cases between this file and k8sresources/modify_test.go. This is
// intentional. However, this test should be used to verify external effects (recording events etc.) that cannot be
// covered modify_test.go, while more fine-grained test cases and variations should rather be added to
// k8sresources/modify_test.go.

var _ = Describe("Dash0 Webhook", func() {
	AfterEach(func() {
		_ = k8sClient.Delete(ctx, BasicCronJob(TestNamespaceName, CronJobName))
		_ = k8sClient.Delete(ctx, BasicDaemonSet(TestNamespaceName, DaemonSetName))
		_ = k8sClient.Delete(ctx, BasicDeployment(TestNamespaceName, DeploymentName))
		_ = k8sClient.Delete(ctx, BasicJob(TestNamespaceName, JobName))
		err := k8sClient.Delete(ctx, BasicReplicaSet(TestNamespaceName, ReplicaSetName))
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "cannot delete replicaset: %v\n", err)
		}
		_ = k8sClient.Delete(ctx, BasicStatefulSet(TestNamespaceName, StatefulSetName))

	})

	Context("when mutating new deployments", func() {
		It("should inject Dash0 into a new basic deployment", func() {
			CreateBasicDeployment(ctx, k8sClient, TestNamespaceName, DeploymentName)
			deployment := GetDeployment(ctx, k8sClient, TestNamespaceName, DeploymentName)
			VerifyModifiedDeployment(deployment, BasicPodSpecExpectations)
		})

		It("should inject Dash0 into a new deployment that has multiple containers, and already has volumes and init containers", func() {
			deployment := DeploymentWithMoreBellsAndWhistles(TestNamespaceName, DeploymentName)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			deployment = GetDeployment(ctx, k8sClient, TestNamespaceName, DeploymentName)
			VerifyModifiedDeployment(deployment, PodSpecExpectations{
				Volumes:               3,
				Dash0VolumeIdx:        2,
				InitContainers:        3,
				Dash0InitContainerIdx: 2,
				Containers: []ContainerExpectations{
					{
						VolumeMounts:                             2,
						Dash0VolumeMountIdx:                      1,
						EnvVars:                                  3,
						NodeOptionsEnvVarIdx:                     1,
						Dash0CollectorBaseUrlEnvVarIdx:           2,
						Dash0CollectorBaseUrlEnvVarExpectedValue: "http://dash0-opentelemetry-collector-daemonset.test-namespace.svc.cluster.local:4318",
					},
					{
						VolumeMounts:                             3,
						Dash0VolumeMountIdx:                      2,
						EnvVars:                                  4,
						NodeOptionsEnvVarIdx:                     2,
						Dash0CollectorBaseUrlEnvVarIdx:           3,
						Dash0CollectorBaseUrlEnvVarExpectedValue: "http://dash0-opentelemetry-collector-daemonset.test-namespace.svc.cluster.local:4318",
					},
				},
			})
			VerifySuccessEvent(ctx, clientset, TestNamespaceName, DeploymentName, "webhook")
		})

		It("should update existing Dash0 artifacts in a new deployment", func() {
			deployment := DeploymentWithExistingDash0Artifacts(TestNamespaceName, DeploymentName)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			deployment = GetDeployment(ctx, k8sClient, TestNamespaceName, DeploymentName)
			VerifyModifiedDeployment(deployment, PodSpecExpectations{
				Volumes:               3,
				Dash0VolumeIdx:        1,
				InitContainers:        3,
				Dash0InitContainerIdx: 1,
				Containers: []ContainerExpectations{
					{
						VolumeMounts:                             2,
						Dash0VolumeMountIdx:                      1,
						EnvVars:                                  3,
						NodeOptionsEnvVarIdx:                     1,
						NodeOptionsUsesValueFrom:                 true,
						Dash0CollectorBaseUrlEnvVarIdx:           2,
						Dash0CollectorBaseUrlEnvVarExpectedValue: "http://dash0-opentelemetry-collector-daemonset.test-namespace.svc.cluster.local:4318",
					},
					{
						VolumeMounts:                             3,
						Dash0VolumeMountIdx:                      1,
						EnvVars:                                  3,
						NodeOptionsEnvVarIdx:                     1,
						NodeOptionsValue:                         "--require /opt/dash0/instrumentation/node.js/node_modules/@dash0/opentelemetry/src/index.js --require something-else --experimental-modules",
						Dash0CollectorBaseUrlEnvVarIdx:           0,
						Dash0CollectorBaseUrlEnvVarExpectedValue: "http://dash0-opentelemetry-collector-daemonset.test-namespace.svc.cluster.local:4318",
					},
				},
			})
		})

		It("should inject Dash0 into a new basic cron job", func() {
			CreateBasicCronJob(ctx, k8sClient, TestNamespaceName, CronJobName)
			cronJob := GetCronJob(ctx, k8sClient, TestNamespaceName, CronJobName)
			VerifyModifiedCronJob(cronJob, BasicPodSpecExpectations)
		})

		It("should inject Dash0 into a new basic daemon set", func() {
			CreateBasicDaemonSet(ctx, k8sClient, TestNamespaceName, DaemonSetName)
			daemonSet := GetDaemonSet(ctx, k8sClient, TestNamespaceName, DaemonSetName)
			VerifyModifiedDaemonSet(daemonSet, BasicPodSpecExpectations)
		})

		It("should inject Dash0 into a new basic job", func() {
			CreateBasicJob(ctx, k8sClient, TestNamespaceName, JobName)
			job := GetJob(ctx, k8sClient, TestNamespaceName, JobName)
			VerifyModifiedJob(job, BasicPodSpecExpectations)
		})

		It("should inject Dash0 into a new basic replica set", func() {
			CreateBasicReplicaSet(ctx, k8sClient, TestNamespaceName, ReplicaSetName)
			replicaSet := GetReplicaSet(ctx, k8sClient, TestNamespaceName, ReplicaSetName)
			VerifyModifiedReplicaSet(replicaSet, BasicPodSpecExpectations)
		})

		It("should not inject Dash0 into a new replica set owned by a deployment", func() {
			CreateReplicaSetOwnedByDeployment(ctx, k8sClient, TestNamespaceName, ReplicaSetName)
			replicaSet := GetReplicaSet(ctx, k8sClient, TestNamespaceName, ReplicaSetName)
			VerifyUnmodifiedReplicaSet(replicaSet)
		})

		It("should inject Dash0 into a new basic stateful set", func() {
			CreateBasicStatefulSet(ctx, k8sClient, TestNamespaceName, StatefulSetName)
			statefulSet := GetStatefulSet(ctx, k8sClient, TestNamespaceName, StatefulSetName)
			VerifyModifiedStatefulSet(statefulSet, BasicPodSpecExpectations)
		})
	})
})
