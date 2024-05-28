// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package k8sresources

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/dash0hq/dash0-operator/internal/util"

	. "github.com/dash0hq/dash0-operator/test/util"
)

// Maintenance note: There is some overlap of test cases between this file and dash0_webhook_test.go. This is
// intentional. However, this test should be used for more fine-grained test cases, while dash0_webhook_test.go should
// be used to verify external effects (recording events etc.) that cannot be covered in this test.

var (
	instrumentationMetadata = util.InstrumentationMetadata{
		Versions: util.Versions{OperatorVersion: "1.2.3",
			InitContainerImageVersion: "4.5.6",
		},
		InstrumentedBy: "modify_test",
	}
)

var _ = Describe("Dash0 Resource Modification", func() {

	ctx := context.Background()
	logger := log.FromContext(ctx)
	resourceModifier := NewResourceModifier(instrumentationMetadata, &logger)

	Context("when instrumenting resources", func() {
		It("should add Dash0 to a basic deployment", func() {
			deployment := BasicDeployment(TestNamespaceName, DeploymentName)
			result := resourceModifier.ModifyDeployment(deployment, TestNamespaceName)

			Expect(result).To(BeTrue())
			VerifyModifiedDeployment(deployment, BasicInstrumentedPodSpecExpectations)
		})

		It("should add Dash0 to a deployment that has multiple containers, and already has volumes and init containers", func() {
			deployment := DeploymentWithMoreBellsAndWhistles(TestNamespaceName, DeploymentName)
			result := resourceModifier.ModifyDeployment(deployment, TestNamespaceName)

			Expect(result).To(BeTrue())
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
		})

		It("should update existing Dash0 artifacts in a deployment", func() {
			deployment := DeploymentWithExistingDash0Artifacts(TestNamespaceName, DeploymentName)
			result := resourceModifier.ModifyDeployment(deployment, TestNamespaceName)

			Expect(result).To(BeTrue())
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

		It("should add Dash0 to a basic cron job", func() {
			resource := BasicCronJob(TestNamespaceName, CronJobName)
			result := resourceModifier.ModifyCronJob(resource, TestNamespaceName)

			Expect(result).To(BeTrue())
			VerifyModifiedCronJob(resource, BasicInstrumentedPodSpecExpectations)
		})

		It("should add Dash0 to a basic daemon set", func() {
			resource := BasicDaemonSet(TestNamespaceName, DaemonSetName)
			result := resourceModifier.ModifyDaemonSet(resource, TestNamespaceName)

			Expect(result).To(BeTrue())
			VerifyModifiedDaemonSet(resource, BasicInstrumentedPodSpecExpectations)
		})

		It("should add Dash0 to a basic job", func() {
			resource := BasicJob(TestNamespaceName, JobName1)
			result := resourceModifier.ModifyJob(resource, TestNamespaceName)

			Expect(result).To(BeTrue())
			VerifyModifiedJob(resource, BasicInstrumentedPodSpecExpectations)
		})

		It("should add Dash0 to a basic replica set", func() {
			resource := BasicReplicaSet(TestNamespaceName, ReplicaSetName)
			result := resourceModifier.ModifyReplicaSet(resource, TestNamespaceName)

			Expect(result).To(BeTrue())
			VerifyModifiedReplicaSet(resource, BasicInstrumentedPodSpecExpectations)
		})

		It("should not add Dash0 to a basic replica set that is owned by a deployment", func() {
			resource := ReplicaSetOwnedByDeployment(TestNamespaceName, ReplicaSetName)
			result := resourceModifier.ModifyReplicaSet(resource, TestNamespaceName)

			Expect(result).To(BeFalse())
			VerifyUnmodifiedReplicaSet(resource)
		})

		It("should add Dash0 to a basic stateful set", func() {
			resource := BasicStatefulSet(TestNamespaceName, StatefulSetName)
			result := resourceModifier.ModifyStatefulSet(resource, TestNamespaceName)

			Expect(result).To(BeTrue())
			VerifyModifiedStatefulSet(resource, BasicInstrumentedPodSpecExpectations)
		})
	})

	Context("when reverting resources", func() {
		It("should remove Dash0 from an instrumented deployment", func() {
			deployment := InstrumentedDeployment(TestNamespaceName, DeploymentName)
			result := resourceModifier.RevertDeployment(deployment)

			Expect(result).To(BeTrue())
			VerifyUnmodifiedDeployment(deployment)
		})

		It("should only remove labels from deployment that has dash0.instrumented=false", func() {
			deployment := DeploymentWithInstrumentedFalseLabel(TestNamespaceName, DeploymentName)
			result := resourceModifier.RevertDeployment(deployment)

			Expect(result).To(BeTrue())
			VerifyUnmodifiedDeployment(deployment)
		})

		It("should remove Dash0 from a instrumented deployment that has multiple containers, and already has volumes and init containers previous to being instrumented", func() {
			deployment := InstrumentedDeploymentWithMoreBellsAndWhistles(TestNamespaceName, DeploymentName)
			result := resourceModifier.RevertDeployment(deployment)

			Expect(result).To(BeTrue())
			VerifyRevertedDeployment(deployment, PodSpecExpectations{
				Volumes:               2,
				Dash0VolumeIdx:        -1,
				InitContainers:        2,
				Dash0InitContainerIdx: -1,
				Containers: []ContainerExpectations{
					{
						VolumeMounts:                   1,
						Dash0VolumeMountIdx:            -1,
						EnvVars:                        1,
						NodeOptionsEnvVarIdx:           -1,
						Dash0CollectorBaseUrlEnvVarIdx: -1,
					},
					{
						VolumeMounts:                   2,
						Dash0VolumeMountIdx:            -1,
						EnvVars:                        2,
						NodeOptionsEnvVarIdx:           -1,
						Dash0CollectorBaseUrlEnvVarIdx: -1,
					},
				},
			})
		})

		It("should remove Dash0 from an instrumented cron job", func() {
			resource := InstrumentedCronJob(TestNamespaceName, CronJobName)
			result := resourceModifier.RevertCronJob(resource)

			Expect(result).To(BeTrue())
			VerifyUnmodifiedCronJob(resource)
		})

		It("should remove Dash0 from an instrumented daemon set", func() {
			resource := InstrumentedDaemonSet(TestNamespaceName, DaemonSetName)
			result := resourceModifier.RevertDaemonSet(resource)

			Expect(result).To(BeTrue())
			VerifyUnmodifiedDaemonSet(resource)
		})

		It("should remove Dash0 from an instrumented job", func() {
			resource := InstrumentedJob(TestNamespaceName, JobName1)
			result := resourceModifier.RevertJob(resource)

			Expect(result).To(BeTrue())
			VerifyUnmodifiedJob(resource)
		})

		It("should remove Dash0 from an instrumented replica set", func() {
			resource := InstrumentedReplicaSet(TestNamespaceName, ReplicaSetName)
			result := resourceModifier.RevertReplicaSet(resource)

			Expect(result).To(BeTrue())
			VerifyUnmodifiedReplicaSet(resource)
		})

		It("should not remove Dash0 from a replica set that is owned by a deployment", func() {
			resource := InstrumentedReplicaSetOwnedByDeployment(TestNamespaceName, ReplicaSetName)
			result := resourceModifier.RevertReplicaSet(resource)

			Expect(result).To(BeFalse())
			VerifyModifiedReplicaSet(resource, BasicInstrumentedPodSpecExpectations)
		})

		It("should remove Dash0 from an instrumented stateful set", func() {
			resource := InstrumentedStatefulSet(TestNamespaceName, StatefulSetName)
			result := resourceModifier.RevertStatefulSet(resource)

			Expect(result).To(BeTrue())
			VerifyUnmodifiedStatefulSet(resource)
		})
	})
})