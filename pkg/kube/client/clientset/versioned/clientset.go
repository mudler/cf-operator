/*

Don't alter this file, it was generated.

*/
// Code generated by client-gen. DO NOT EDIT.

package versioned

import (
	boshdeploymentcontrollerv1alpha1 "code.cloudfoundry.org/cf-operator/pkg/kube/client/clientset/versioned/typed/boshdeploymentcontroller/v1alpha1"
	extendedstatefulsetcontrollerv1alpha1 "code.cloudfoundry.org/cf-operator/pkg/kube/client/clientset/versioned/typed/extendedstatefulsetcontroller/v1alpha1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	BoshdeploymentcontrollerV1alpha1() boshdeploymentcontrollerv1alpha1.BoshdeploymentcontrollerV1alpha1Interface
	// Deprecated: please explicitly pick a version if possible.
	Boshdeploymentcontroller() boshdeploymentcontrollerv1alpha1.BoshdeploymentcontrollerV1alpha1Interface
	ExtendedstatefulsetcontrollerV1alpha1() extendedstatefulsetcontrollerv1alpha1.ExtendedstatefulsetcontrollerV1alpha1Interface
	// Deprecated: please explicitly pick a version if possible.
	Extendedstatefulsetcontroller() extendedstatefulsetcontrollerv1alpha1.ExtendedstatefulsetcontrollerV1alpha1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	boshdeploymentcontrollerV1alpha1      *boshdeploymentcontrollerv1alpha1.BoshdeploymentcontrollerV1alpha1Client
	extendedstatefulsetcontrollerV1alpha1 *extendedstatefulsetcontrollerv1alpha1.ExtendedstatefulsetcontrollerV1alpha1Client
}

// BoshdeploymentcontrollerV1alpha1 retrieves the BoshdeploymentcontrollerV1alpha1Client
func (c *Clientset) BoshdeploymentcontrollerV1alpha1() boshdeploymentcontrollerv1alpha1.BoshdeploymentcontrollerV1alpha1Interface {
	return c.boshdeploymentcontrollerV1alpha1
}

// Deprecated: Boshdeploymentcontroller retrieves the default version of BoshdeploymentcontrollerClient.
// Please explicitly pick a version.
func (c *Clientset) Boshdeploymentcontroller() boshdeploymentcontrollerv1alpha1.BoshdeploymentcontrollerV1alpha1Interface {
	return c.boshdeploymentcontrollerV1alpha1
}

// ExtendedstatefulsetcontrollerV1alpha1 retrieves the ExtendedstatefulsetcontrollerV1alpha1Client
func (c *Clientset) ExtendedstatefulsetcontrollerV1alpha1() extendedstatefulsetcontrollerv1alpha1.ExtendedstatefulsetcontrollerV1alpha1Interface {
	return c.extendedstatefulsetcontrollerV1alpha1
}

// Deprecated: Extendedstatefulsetcontroller retrieves the default version of ExtendedstatefulsetcontrollerClient.
// Please explicitly pick a version.
func (c *Clientset) Extendedstatefulsetcontroller() extendedstatefulsetcontrollerv1alpha1.ExtendedstatefulsetcontrollerV1alpha1Interface {
	return c.extendedstatefulsetcontrollerV1alpha1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.boshdeploymentcontrollerV1alpha1, err = boshdeploymentcontrollerv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.extendedstatefulsetcontrollerV1alpha1, err = extendedstatefulsetcontrollerv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.boshdeploymentcontrollerV1alpha1 = boshdeploymentcontrollerv1alpha1.NewForConfigOrDie(c)
	cs.extendedstatefulsetcontrollerV1alpha1 = extendedstatefulsetcontrollerv1alpha1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.boshdeploymentcontrollerV1alpha1 = boshdeploymentcontrollerv1alpha1.New(c)
	cs.extendedstatefulsetcontrollerV1alpha1 = extendedstatefulsetcontrollerv1alpha1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}