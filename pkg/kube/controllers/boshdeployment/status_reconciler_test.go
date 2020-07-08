package boshdeployment_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bdv1 "code.cloudfoundry.org/quarks-operator/pkg/kube/apis/boshdeployment/v1alpha1"
	qstsv1a1 "code.cloudfoundry.org/quarks-operator/pkg/kube/apis/quarksstatefulset/v1alpha1"
	"code.cloudfoundry.org/quarks-operator/pkg/kube/controllers"
	bdplcontroller "code.cloudfoundry.org/quarks-operator/pkg/kube/controllers/boshdeployment"
	cfakes "code.cloudfoundry.org/quarks-operator/pkg/kube/controllers/fakes"
	cfcfg "code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/pointers"
	helper "code.cloudfoundry.org/quarks-utils/testing/testhelper"
)

var _ = Describe("ReconcileBDPL", func() {
	var (
		manager             *cfakes.FakeManager
		reconciler          reconcile.Reconciler
		request             reconcile.Request
		ctx                 context.Context
		log                 *zap.SugaredLogger
		config              *cfcfg.Config
		client              *cfakes.FakeClient
		bdpl                *bdv1.BOSHDeployment
		desiredQStatefulSet *qstsv1a1.QuarksStatefulSet
	)

	BeforeEach(func() {
		controllers.AddToScheme(scheme.Scheme)
		manager = &cfakes.FakeManager{}
		manager.GetSchemeReturns(scheme.Scheme)

		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
		config = &cfcfg.Config{CtxTimeOut: 10 * time.Second}
		_, log = helper.NewTestLogger()
		ctx = ctxlog.NewParentContext(log)

		client = &cfakes.FakeClient{}
		client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
			switch object := object.(type) {
			case *qstsv1a1.QuarksStatefulSet:
				desiredQStatefulSet.DeepCopyInto(object)
				return nil
			case *bdv1.BOSHDeployment:
				bdpl.DeepCopyInto(object)
				return nil
			}

			return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
		})

		manager.GetClientReturns(client)

		client.StatusCalls(func() crc.StatusWriter { return &cfakes.FakeStatusWriter{} })
	})

	JustBeforeEach(func() {
		reconciler = bdplcontroller.NewStatusQSTSReconciler(ctx, config, manager)
		desiredQStatefulSet = &qstsv1a1.QuarksStatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
				UID:       "",
				Labels:    map[string]string{bdv1.LabelDeploymentName: "deployment-name"},
			},
			Spec: qstsv1a1.QuarksStatefulSetSpec{
				Template: appsv1.StatefulSet{
					Spec: appsv1.StatefulSetSpec{
						Replicas: pointers.Int32(1),
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name: "test-container",
								}},
							},
						},
					},
				},
			},
		}
		bdpl = &bdv1.BOSHDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment-name",
				Namespace: "default",
			},
			Spec: bdv1.BOSHDeploymentSpec{
				Manifest: bdv1.ResourceReference{
					Name: "dummy-manifest",
					Type: "configmap",
				},
				Ops: []bdv1.ResourceReference{
					{
						Name: "bar",
						Type: "configmap",
					},
					{
						Name: "baz",
						Type: "secret",
					},
				},
			},
		}
	})

	Context("Provides a quarksStatefulSet definition", func() {

		It("updates new statefulSet and continues to reconcile when new version is not available", func() {

			client.UpdateCalls(func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
				switch bdplUpdate := object.(type) {
				case *bdv1.BOSHDeployment:
					Expect(bdplUpdate.Status.DeployedInstanceGroups).To(Equal(1))
					Expect(bdplUpdate.Name).To(Equal("deployment-name"))
				}
				return nil
			})
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
})
