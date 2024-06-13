// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	operatorNamespace       = "dash0-operator-system"
	operatorImageRepository = "dash0-operator-controller"
	operatorImageTag        = "latest"

	kubeContextForTest            = "docker-desktop"
	applicationUnderTestNamespace = "e2e-application-under-test-namespace"
)

var (
	certManagerHasBeenInstalled = false

	originalKubeContext       string
	kubeContextHasBeenChanged bool

	setupFinishedSuccessfully bool
)

var _ = Describe("Dash0 Kubernetes Operator", Ordered, func() {

	BeforeAll(func() {
		pwdOutput, err := Run(exec.Command("pwd"), false)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		workingDir := strings.TrimSpace(pwdOutput)
		fmt.Fprintf(GinkgoWriter, "workingDir: %s\n", workingDir)

		CheckIfRequiredPortsAreBlocked()
		RenderTemplates()
		kubeContextHasBeenChanged, originalKubeContext = SetKubeContext(kubeContextForTest)

		certManagerHasBeenInstalled = EnsureCertManagerIsInstalled()
		RecreateNamespace(applicationUnderTestNamespace)
		RebuildOperatorControllerImage(operatorImageRepository, operatorImageTag)
		RebuildDash0InstrumentationImage()

		setupFinishedSuccessfully = true
	})

	AfterAll(func() {
		UninstallCertManagerIfApplicable(certManagerHasBeenInstalled)

		if applicationUnderTestNamespace != "default" {
			By("removing namespace for application under test")
			_ = RunAndIgnoreOutput(exec.Command("kubectl", "delete", "ns", applicationUnderTestNamespace))
		}
		if kubeContextHasBeenChanged {
			RevertKubeCtx(originalKubeContext)
		}
	})

	BeforeEach(func() {
		DeleteTestIdFiles()
	})

	AfterEach(func() {
		if setupFinishedSuccessfully {
			// As an alternative to undeploying all applications under test (deployment, daemonset, cronjob, ...) we
			// could also delete the whole namespace for the application under test to after each test case get rid of
			// all applications (and then recreate the namespace before each test). However, this would mean we also
			// need to deploy the OpenTelemetry collector to the target namespace again for each test case, which would
			// slow down tests a bit more.
			RemoveAllTestApplications(applicationUnderTestNamespace)
			DeleteTestIdFiles()
		}
	})

	Describe("controller", func() {
		AfterEach(func() {
			UndeployDash0Resource(applicationUnderTestNamespace)
			UndeployOperatorAndCollector(operatorNamespace)
		})

		type controllerTest struct {
			workloadType        string
			installWorkload     func(string) error
			isBatch             bool
			restartPodsManually bool
		}

		DescribeTable(
			"when instrumenting existing workloads",
			func(config controllerTest) {
				By(fmt.Sprintf("installing the Node.js %s", config.workloadType))
				Expect(config.installWorkload(applicationUnderTestNamespace)).To(Succeed())
				By("deploy the operator and the Dash0 custom resource")

				DeployOperatorWithCollectorAndClearExportedTelemetry(operatorNamespace, operatorImageRepository, operatorImageTag)
				DeployDash0Resource(applicationUnderTestNamespace)
				By(fmt.Sprintf("verifying that the Node.js %s has been instrumented by the controller", config.workloadType))
				testId := VerifyThatWorkloadHasBeenInstrumented(
					applicationUnderTestNamespace,
					config.workloadType,
					config.isBatch,
					config.restartPodsManually,
					"controller",
				)

				UndeployDash0Resource(applicationUnderTestNamespace)

				VerifyThatInstrumentationHasBeenReverted(
					applicationUnderTestNamespace,
					config.workloadType,
					config.isBatch,
					config.restartPodsManually,
					testId,
					"controller",
				)
			},
			Entry("should instrument and uninstrument existing cron jobs", controllerTest{
				workloadType:    "cronjob",
				installWorkload: InstallNodeJsCronJob,
				isBatch:         true,
			}),
			Entry("should instrument and uninstrument existing daemon sets", controllerTest{
				workloadType:    "daemonset",
				installWorkload: InstallNodeJsDaemonSet,
			}),
			Entry("should instrument and uninstrument existing deployments", controllerTest{
				workloadType:    "deployment",
				installWorkload: InstallNodeJsDeployment,
			}),
			Entry("should instrument and uninstrument existing replica set", controllerTest{
				workloadType:        "replicaset",
				installWorkload:     InstallNodeJsReplicaSet,
				restartPodsManually: true,
			}),
			Entry("should instrument and uninstrument existing stateful set", controllerTest{
				workloadType:    "statefulset",
				installWorkload: InstallNodeJsStatefulSet,
			}),
		)

		Describe("when it detects existing immutable jobs", func() {
			It("should label them accordingly", func() {
				By("installing the Node.js job")
				Expect(InstallNodeJsJob(applicationUnderTestNamespace)).To(Succeed())
				By("deploy the operator and the Dash0 custom resource")
				DeployOperatorWithCollectorAndClearExportedTelemetry(operatorNamespace, operatorImageRepository, operatorImageTag)
				DeployDash0Resource(applicationUnderTestNamespace)
				By("verifying that the Node.js job has been labelled by the controller and that an event has been emitted")
				Eventually(func(g Gomega) {
					VerifyLabels(g, applicationUnderTestNamespace, "job", false, "controller")
					VerifyFailedInstrumentationEvent(
						g,
						applicationUnderTestNamespace,
						"job",
						"Dash0 instrumentation of this workload by the controller has not been successful. "+
							"Error message: Dash0 cannot instrument the existing job "+
							"e2e-application-under-test-namespace/dash0-operator-nodejs-20-express-test-job, since "+
							"this type of workload is immutable.",
					)
				}, verifyTelemetryTimeout, verifyTelemetryPollingInterval).Should(Succeed())

				UndeployDash0Resource(applicationUnderTestNamespace)

				VerifyThatFailedInstrumentationAttemptLabelsHaveBeenRemovedRemoved(applicationUnderTestNamespace, "job")
			})
		})
	})

	Describe("webhook", func() {

		BeforeAll(func() {
			DeployOperatorWithCollectorAndClearExportedTelemetry(operatorNamespace, operatorImageRepository, operatorImageTag)
			DeployDash0Resource(applicationUnderTestNamespace)

			fmt.Fprint(GinkgoWriter, "waiting 10 seconds to give the webhook some time to get ready\n")
			time.Sleep(10 * time.Second)
		})

		AfterAll(func() {
			UndeployDash0Resource(applicationUnderTestNamespace)
			UndeployOperatorAndCollector(operatorNamespace)
		})

		type webhookTest struct {
			workloadType    string
			installWorkload func(string) error
			isBatch         bool
		}

		DescribeTable(
			"when instrumenting new workloads",
			func(config webhookTest) {
				By(fmt.Sprintf("installing the Node.js %s", config.workloadType))
				Expect(config.installWorkload(applicationUnderTestNamespace)).To(Succeed())
				By(fmt.Sprintf("verifying that the Node.js %s has been instrumented by the webhook", config.workloadType))
				VerifyThatWorkloadHasBeenInstrumented(
					applicationUnderTestNamespace,
					config.workloadType,
					config.isBatch,
					false,
					"webhook",
				)

				if config.workloadType == "job" {
					// For all other workload types, reverting the instrumentation is tested by the controller test
					// suite. But the controller cannot instrument jobs, so we cannot test the (failing)
					// uninstrumentation procedure there. Thus, for jobs, we test the failing uninstrumentation and
					// its effects here.
					UndeployDash0Resource(applicationUnderTestNamespace)

					Eventually(func(g Gomega) {
						// Verify that the instrumentation labels are still in place -- since we cannot undo the
						// instrumentation, the labels must also not be removed.
						VerifyLabels(g, applicationUnderTestNamespace, config.workloadType, true, "webhook")
						VerifyFailedUninstrumentationEvent(
							g,
							applicationUnderTestNamespace,
							"job",
							"The controller's attempt to remove the Dash0 instrumentation from this workload has not "+
								"been successful. Error message: Dash0 cannot remove the instrumentation from the "+
								"existing job "+
								"e2e-application-under-test-namespace/dash0-operator-nodejs-20-express-test-job, since "+
								"this type of workload is immutable.",
						)
					}, verifyTelemetryTimeout, verifyTelemetryPollingInterval).Should(Succeed())
				}
			},
			Entry("should instrument new cron jobs", webhookTest{
				workloadType:    "cronjob",
				installWorkload: InstallNodeJsCronJob,
				isBatch:         true,
			}),
			Entry("should instrument new daemon sets", webhookTest{
				workloadType:    "daemonset",
				installWorkload: InstallNodeJsDaemonSet,
			}),
			Entry("should instrument new deployments", webhookTest{
				workloadType:    "deployment",
				installWorkload: InstallNodeJsDeployment,
			}),
			Entry("should instrument new jobs", webhookTest{
				workloadType:    "job",
				installWorkload: InstallNodeJsJob,
				isBatch:         true,
			}),
			Entry("should instrument new replica sets", webhookTest{
				workloadType:    "replicaset",
				installWorkload: InstallNodeJsReplicaSet,
			}),
			Entry("should instrument new stateful sets", webhookTest{
				workloadType:    "statefulset",
				installWorkload: InstallNodeJsStatefulSet,
			}),
		)
	})
})
