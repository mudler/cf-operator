package extendedstatefulset

import (
	"go.uber.org/zap"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	essv1 "code.cloudfoundry.org/cf-operator/pkg/kube/apis/extendedstatefulset/v1alpha1"
)

// Add creates a new ExtendedStatefulSet controller and adds it to the Manager
func Add(log *zap.SugaredLogger, mgr manager.Manager) error {
	r := NewReconciler(log, mgr, controllerutil.SetControllerReference)

	// Create a new controller
	c, err := controller.New("extendedstatefulset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ExtendedStatefulSet
	err = c.Watch(&source.Kind{Type: &essv1.ExtendedStatefulSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}
