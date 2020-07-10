package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bdv1 "code.cloudfoundry.org/quarks-operator/pkg/kube/apis/boshdeployment/v1alpha1"
	qstsv1a1 "code.cloudfoundry.org/quarks-operator/pkg/kube/apis/quarksstatefulset/v1alpha1"
	bm "code.cloudfoundry.org/quarks-operator/testing/boshmanifest"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

var _ = Describe("Deploy", func() {
	const (
		deploymentName = "test"
		manifestName   = "manifest"

		opReplaceEnv = `- type: replace
  path: /instance_groups/name=quarks-gora/jobs/name=quarks-gora/properties/quarks?
  value:
    envs:
    - name: XPOD_IPX
      valueFrom:
        fieldRef:
          apiVersion: v1
          fieldPath: status.podIP
`

		opReadiness = `- type: replace
  path: /instance_groups/name=quarks-gora?/jobs/name=quarks-gora?/properties/quarks/run
  value:
    healthcheck:
      quarks-gora:
        readiness:
          exec:
            command:
            - "echo healthy"
`

		opOneInstance = `- type: replace
  path: /instance_groups/name=quarks-gora?/instances
  value: 1
`
		opInstances = `- type: replace
  path: /instance_groups/name=quarks-gora?/instances
  value: 3
`
		opRemoveApi = `- type: remove
  path: /instance_groups/name=api
`
	)

	var tearDowns []machine.TearDownFunc

	AfterEach(func() {
		Expect(env.TearDownAll(tearDowns)).To(Succeed())
	})

	Context("when using the default configuration", func() {
		const (
			headlessSvcName  = "quarks-gora"
			clusterIpSvcName = "quarks-gora-0"
		)

		It("should deploy a pod and create services", func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, env.BOSHManifestConfigMap(manifestName, bm.Gora))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment(deploymentName, manifestName))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for instance group pods")
			err = env.WaitForInstanceGroup(env.Namespace, deploymentName, "quarks-gora", "1", 2)
			Expect(err).NotTo(HaveOccurred(), "error waiting for instance group pods from deployment")

			By("checking for services")
			svc, err := env.GetService(env.Namespace, headlessSvcName)
			Expect(err).NotTo(HaveOccurred(), "error getting service for instance group")
			Expect(svc.Spec.Selector).To(Equal(map[string]string{bdv1.LabelInstanceGroupName: "quarks-gora", bdv1.LabelDeploymentName: deploymentName}))
			Expect(svc.Spec.Ports).NotTo(BeEmpty())
			Expect(svc.Spec.Ports[0].Name).To(Equal("quarks-gora"))
			Expect(svc.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(4222)))

			svc, err = env.GetService(env.Namespace, clusterIpSvcName)
			Expect(err).NotTo(HaveOccurred(), "error getting service for instance group")
			Expect(svc.Spec.Selector).To(Equal(map[string]string{
				bdv1.LabelInstanceGroupName: "quarks-gora",
				qstsv1a1.LabelAZIndex:       "0",
				qstsv1a1.LabelPodOrdinal:    "0",
				bdv1.LabelDeploymentName:    deploymentName,
			}))
			Expect(svc.Spec.Ports).NotTo(BeEmpty())
			Expect(svc.Spec.Ports[0].Name).To(Equal("quarks-gora"))
			Expect(svc.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(4222)))
		})

		It("should deploy manifest with multiple ops correctly", func() {
			var bdpl *bdv1.BOSHDeployment
			tearDown, err := env.CreateConfigMap(env.Namespace, env.BOSHManifestConfigMap(manifestName, bm.Gora))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			tearDown, err = env.CreateConfigMap(env.Namespace, env.CustomOpsConfigMap("bosh-ops", opOneInstance))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			tearDown, err = env.CreateSecret(env.Namespace, env.CustomOpsSecret("bosh-ops-secret", opInstances))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.InterpolateBOSHDeployment(deploymentName, manifestName, "bosh-ops", "bosh-ops-secret"))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for instance group pods")
			err = env.WaitForInstanceGroup(env.Namespace, deploymentName, "quarks-gora", "1", 3)
			Expect(err).NotTo(HaveOccurred(), "error waiting for instance group pods from deployment")

			sts, err := env.GetStatefulSet(env.Namespace, "quarks-gora")
			Expect(err).NotTo(HaveOccurred(), "error getting statefulset for deployment")
			Expect(*sts.Spec.Replicas).To(BeEquivalentTo(3))

			bdpl, err = env.GetBOSHDeployment(env.Namespace, deploymentName)
			Expect(err).NotTo(HaveOccurred())
			Expect(bdpl.Status.DeployedInstanceGroups).To(BeEquivalentTo(1))
			bdpl, err = env.GetBOSHDeployment(env.Namespace, deploymentName)

			Expect(err).NotTo(HaveOccurred())
			Expect(bdpl.Status.TotalInstanceGroups).To(BeEquivalentTo(1))

			Expect(err).NotTo(HaveOccurred())
			Expect(bdpl.Status.TotalJobCount).To(BeEquivalentTo(2))
		})

	})

	Context("when using pre-render scripts", func() {
		podName := "nats-0"

		It("it should run them", func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: manifestName},
				Data: map[string]string{
					"manifest": bm.NatsSmallWithPatch,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment(deploymentName, manifestName))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for init container")

			err = env.WaitForInitContainerRunning(env.Namespace, podName, "bosh-pre-start-nats")
			Expect(err).NotTo(HaveOccurred(), "error waiting for pre-start init container from pod")

			Expect(env.WaitForPodContainerLogMsg(env.Namespace, podName, "bosh-pre-start-nats", "this file was patched")).To(BeNil(), "error getting logs from drain_watch process")
		})
	})

	Context("when BPM has pre-start hooks configured", func() {
		It("should run pre-start script in an init container", func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "diego-manifest"},
				Data:       map[string]string{"manifest": bm.Diego},
			})
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment("diego", "diego-manifest"))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for instance group pods")
			err = env.WaitForInstanceGroup(env.Namespace, "diego", "file_server", "1", 2)
			Expect(err).NotTo(HaveOccurred(), "error waiting for instance group pods from deployment")

			By("checking for containers")
			pods, _ := env.GetPods(env.Namespace, "quarks.cloudfoundry.org/instance-group-name=file_server")
			Expect(len(pods.Items)).To(Equal(2))
			pod := pods.Items[1]
			Expect(pod.Spec.InitContainers).To(HaveLen(6))
			Expect(pod.Spec.InitContainers[5].Command).To(Equal([]string{"/usr/bin/dumb-init", "--"}))
			Expect(pod.Spec.InitContainers[5].Args).To(Equal([]string{
				"/bin/sh",
				"-xc",
				"time /var/vcap/jobs/file_server/bin/bpm-pre-start",
			}))
		})
	})

	Context("when BOSH has pre-start hooks configured", func() {
		It("should run pre-start script in an init container", func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "cfrouting-manifest"},
				Data:       map[string]string{"manifest": bm.CFRouting},
			})
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment("bph", "cfrouting-manifest"))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for instance group pods")
			err = env.WaitForInstanceGroup(env.Namespace, "bph", "route_registrar", "1", 2)
			Expect(err).NotTo(HaveOccurred(), "error waiting for instance group pods from deployment")

			By("checking for containers")
			pods, _ := env.GetPods(env.Namespace, "quarks.cloudfoundry.org/instance-group-name=route_registrar")
			Expect(len(pods.Items)).To(Equal(2))

			pod := pods.Items[1]
			Expect(pod.Spec.InitContainers).To(HaveLen(5))
			Expect(pod.Spec.InitContainers[4].Name).To(Equal("bosh-pre-start-route-registrar"))
		})
	})

	Context("when job name contains an underscore", func() {
		It("should apply naming guidelines", func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "router-manifest"},
				Data:       map[string]string{"manifest": bm.CFRouting},
			})
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment("routing", "router-manifest"))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for instance group pods")
			err = env.WaitForInstanceGroup(env.Namespace, "routing", "route_registrar", "1", 2)
			Expect(err).NotTo(HaveOccurred(), "error waiting for instance group pods from deployment")

			By("checking for containers")
			pods, _ := env.GetPods(env.Namespace, "quarks.cloudfoundry.org/instance-group-name=route_registrar")
			Expect(len(pods.Items)).To(Equal(2))
			Expect(pods.Items[0].Spec.Containers).To(HaveLen(2))
			Expect(pods.Items[0].Spec.Containers[0].Name).To(Equal("route-registrar-route-registrar"))
		})
	})

	Context("when updating a readiness probe", func() {
		BeforeEach(func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, env.BOSHManifestConfigMap(manifestName, bm.Gora))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment(deploymentName, manifestName))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for instance group pods")
			err = env.WaitForPods(env.Namespace, "quarks.cloudfoundry.org/instance-group-name=quarks-gora")
			Expect(err).NotTo(HaveOccurred(), "error waiting for instance group pods from deployment")
		})

		Context("by adding an ops file to the bdpl custom resource", func() {
			It("should update the deployment and respect the instance pods", func() {
				tearDown, err := env.CreateConfigMap(env.Namespace, env.CustomOpsConfigMap("readiness-ops", opReadiness))
				Expect(err).NotTo(HaveOccurred())
				tearDowns = append(tearDowns, tearDown)

				bdpl, err := env.GetBOSHDeployment(env.Namespace, deploymentName)
				Expect(err).NotTo(HaveOccurred())
				bdpl.Spec.Ops = []bdv1.ResourceReference{{Name: "readiness-ops", Type: bdv1.ConfigMapReference}}

				_, _, err = env.UpdateBOSHDeployment(env.Namespace, *bdpl)
				Expect(err).NotTo(HaveOccurred())

				By("checking for statefulset")
				Eventually(func() []string {
					sts, err := env.GetStatefulSet(env.Namespace, "quarks-gora")
					Expect(err).NotTo(HaveOccurred(), "error getting statefulset for deployment")

					if sts.Spec.Template.Spec.Containers[0].ReadinessProbe != nil {
						return sts.Spec.Template.Spec.Containers[0].ReadinessProbe.Exec.Command
					}
					return []string{}
				}, env.PollTimeout, env.PollInterval).Should(Equal([]string{"echo healthy"}), "command for readiness should be created")
			})
		})
	})

	Context("when ops file is adding env vars", func() {
		BeforeEach(func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, env.BOSHManifestConfigMap(manifestName, bm.Gora))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			tearDown, err = env.CreateConfigMap(env.Namespace, env.CustomOpsConfigMap("ops-bpm", opReplaceEnv))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeploymentWithOps(deploymentName, manifestName, "ops-bpm"))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("checking for instance group pods")
			err = env.WaitForInstanceGroup(env.Namespace, deploymentName, "quarks-gora", "1", 2)
			Expect(err).NotTo(HaveOccurred(), "error waiting for instance group pods from deployment")
		})

		It("should add the env var to the container", func() {
			pod, err := env.GetPod(env.Namespace, "quarks-gora-1")
			Expect(err).NotTo(HaveOccurred())

			Expect(env.EnvKeys(pod.Spec.Containers)).To(ContainElement("XPOD_IPX"))
		})
	})

	Context("when using a custom reconciler configuration", func() {
		It("should use the context timeout (1ns)", func() {
			env.Config.CtxTimeOut = 1 * time.Nanosecond
			defer func() {
				env.Config.CtxTimeOut = 10 * time.Second
			}()

			tearDown, err := env.CreateConfigMap(env.Namespace, env.BOSHManifestConfigMap(manifestName, bm.Gora))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment(deploymentName, manifestName))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			Expect(env.WaitForLogMsg(env.ObservedLogs, "context deadline exceeded")).To(Succeed())
		})
	})

	Context("when data provided by the user is incorrect", func() {
		BeforeEach(func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, env.BOSHManifestConfigMap(manifestName, bm.Gora))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
		})

		It("fails to create the resource if the validator gets an error when applying ops files", func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, env.CustomOpsConfigMap("bosh-ops", opOneInstance))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			tearDown, err = env.CreateSecret(env.Namespace, env.CustomOpsSecret("bosh-ops-secret", opRemoveApi))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.InterpolateBOSHDeployment(deploymentName, manifestName, "bosh-ops", "bosh-ops-secret"))
			Expect(err).To(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
			Expect(err.Error()).To(ContainSubstring(`admission webhook "validate-boshdeployment.quarks.cloudfoundry.org" denied the request:`))
		})

		It("failed to deploy an empty manifest", func() {
			_, tearDown, err := env.CreateBOSHDeployment(env.Namespace, env.EmptyBOSHDeployment(deploymentName, manifestName))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(SatisfyAny(
				ContainSubstring("spec.manifest.type in body should be one of"),
				// kube > 1.16
				ContainSubstring("spec.manifest.type: Unsupported value"),
			))
			Expect(err.Error()).To(ContainSubstring("spec.manifest.name in body should be at least 1 chars long"))
			tearDowns = append(tearDowns, tearDown)
		})

		It("failed to deploy due to a wrong manifest type", func() {
			_, tearDown, err := env.CreateBOSHDeployment(env.Namespace, env.WrongTypeBOSHDeployment(deploymentName, manifestName))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(SatisfyAny(
				ContainSubstring("spec.manifest.type in body should be one of"),
				// kube > 1.16
				ContainSubstring("spec.manifest.type: Unsupported value"),
			))
			tearDowns = append(tearDowns, tearDown)
		})

		It("failed to deploy due to an empty manifest ref", func() {
			_, tearDown, err := env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment(deploymentName, ""))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.manifest.name in body should be at least 1 chars long"))
			tearDowns = append(tearDowns, tearDown)
		})

		It("failed to deploy due to a wrong ops type", func() {
			_, tearDown, err := env.CreateBOSHDeployment(env.Namespace, env.BOSHDeploymentWithWrongTypeOps(deploymentName, manifestName, "ops"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(SatisfyAny(
				ContainSubstring("spec.ops.type in body should be one of"),
				// kube > 1.16
				ContainSubstring("spec.ops.type: Unsupported value"),
			))
			tearDowns = append(tearDowns, tearDown)
		})

		It("failed to deploy due to an empty ops ref", func() {
			_, tearDown, err := env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeploymentWithOps(deploymentName, manifestName, ""))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.ops.name in body should be at least 1 chars long"))
			tearDowns = append(tearDowns, tearDown)
		})

		It("failed to deploy due to a not existing ops ref", func() {
			// use a not created configmap name, so that we will hit errors while resources do not exist.
			_, tearDown, err := env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeploymentWithOps(deploymentName, manifestName, "bosh-ops-unknown"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Timeout reached. Resources 'configmap/bosh-ops-unknown' do not exist"))
			tearDowns = append(tearDowns, tearDown)
		})

		It("failed to deploy if the ops resource is not available before timeout", func(done Done) {
			ch := make(chan machine.ChanResult)

			go env.CreateBOSHDeploymentUsingChan(ch, env.Namespace, env.DefaultBOSHDeploymentWithOps(deploymentName, manifestName, "bosh-ops"))

			time.Sleep(8 * time.Second)

			// Generate the right ops resource, so that the above goroutine will not end in error
			_, err := env.CreateConfigMap(env.Namespace, env.CustomOpsConfigMap("bosh-ops", opOneInstance))
			Expect(err).NotTo(HaveOccurred())

			chanReceived := <-ch
			Expect(chanReceived.Error).To(HaveOccurred())
			close(done)
		}, 10)

		It("does not failed to deploy if the ops ref is created on time", func(done Done) {
			ch := make(chan machine.ChanResult)
			go env.CreateBOSHDeploymentUsingChan(ch, env.Namespace, env.DefaultBOSHDeploymentWithOps(deploymentName, manifestName, "bosh-ops"))

			// Generate the right ops resource, so that the above goroutine will not end in error
			tearDown, err := env.CreateConfigMap(env.Namespace, env.CustomOpsConfigMap("bosh-ops", opOneInstance))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			chanReceived := <-ch
			Expect(chanReceived.Error).NotTo(HaveOccurred())
			close(done)
		}, 5)
	})

	Context("when a BOSHDeployment already exists in the namespace", func() {
		BeforeEach(func() {
			tearDown, err := env.CreateConfigMap(env.Namespace, env.BOSHManifestConfigMap(manifestName, bm.Gora))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			_, tearDown, err = env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment(deploymentName, manifestName))
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
		})

		It("fails to create the resource", func() {
			_, _, err := env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment("second-bdpl", manifestName))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`denied the request: Only one deployment allowed`))
		})
	})

	Context("when the BOSHDeployment cannot be resolved", func() {
		It("should not create the resource and the validation hook should return an error message", func() {
			_, _, err := env.CreateBOSHDeployment(env.Namespace, env.DefaultBOSHDeployment(deploymentName, "foo-baz"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`admission webhook "validate-boshdeployment.quarks.cloudfoundry.org" denied the request:`))
			Expect(err.Error()).To(ContainSubstring(`ConfigMap "foo-baz" not found`))
		})
	})
})
