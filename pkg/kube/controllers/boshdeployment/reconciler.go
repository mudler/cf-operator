package boshdeployment

import (
	"context"
	"fmt"
	"strconv"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bdm "code.cloudfoundry.org/cf-operator/pkg/bosh/manifest"
	bdc "code.cloudfoundry.org/cf-operator/pkg/kube/apis/boshdeployment/v1alpha1"
)

// Check that ReconcileBOSHDeployment implements the reconcile.Reconciler interface
var _ reconcile.Reconciler = &ReconcileBOSHDeployment{}

type setReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(log *zap.SugaredLogger, mgr manager.Manager, resolver bdm.Resolver, srf setReferenceFunc) reconcile.Reconciler {
	return &ReconcileBOSHDeployment{
		log:          log,
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		resolver:     resolver,
		setReference: srf,
	}
}

// ReconcileBOSHDeployment reconciles a BOSHDeployment object
type ReconcileBOSHDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	scheme       *runtime.Scheme
	resolver     bdm.Resolver
	setReference setReferenceFunc
	log          *zap.SugaredLogger
}

// Reconcile reads that state of the cluster for a BOSHDeployment object and makes changes based on the state read
// and what is in the BOSHDeployment.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBOSHDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.log.Infof("Reconciling BOSHDeployment %s\n", request.NamespacedName)

	// Fetch the BOSHDeployment instance
	instance := &bdc.BOSHDeployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.log.Debug("Skip reconcile: CRD not found\n")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// retrieve manifest
	manifest, err := r.resolver.ResolveCRD(instance.Spec, request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	// TODO validation
	if len(manifest.InstanceGroups) < 1 {
		return reconcile.Result{}, fmt.Errorf("manifest is missing instance groups")
	}

	// Define a new Pod object
	pod := newPodForCR(manifest, request.Namespace)

	// Set BOSHDeployment instance as the owner and controller
	if err := r.setReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// TODO example implementation, untested, replace eventually
	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.log.Infof("Creating a new Pod %s/%s\n", pod.Namespace, pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	r.log.Infof("Skip reconcile: Pod %s/%s already exists", found.Namespace, found.Name)
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *bdm.Manifest, namespace string) *corev1.Pod {
	t := int64(1)
	ig := cr.InstanceGroups[0]
	labels := map[string]string{
		"app":  ig.Name,
		"size": strconv.Itoa(ig.Instances),
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ig.Name + "-pod",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &t,
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
