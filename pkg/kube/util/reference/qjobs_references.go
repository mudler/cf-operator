package reference

import (
	"context"

	qjv1a1 "code.cloudfoundry.org/quarks-job/pkg/kube/apis/quarksjob/v1alpha1"
	bdv1 "code.cloudfoundry.org/quarks-operator/pkg/kube/apis/boshdeployment/v1alpha1"
	"github.com/pkg/errors"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetQJobsReferencedBy returns a list of all QJob referenced by a BOSHDeployment
// The object can be an QuarksStatefulSet or a BOSHDeployment
func GetQJobsReferencedBy(ctx context.Context, client crc.Client, bdpl bdv1.BOSHDeployment) (map[string]bool, error) {

	bdplQjobs := map[string]bool{}
	list, err := listQjobs(ctx, client, bdpl.GetNamespace())
	if err != nil {
		return nil, errors.Wrap(err, "failed getting QJob List")
	}

	for _, qjob := range list.Items {
		if qjob.GetLabels()[bdv1.LabelDeploymentName] == bdpl.Name {
			bdplQjobs[qjob.GetName()] = qjob.Status.Completed
		}
	}

	return bdplQjobs, nil
}

func listQjobs(ctx context.Context, client crc.Client, namespace string) (*qjv1a1.QuarksJobList, error) {
	result := &qjv1a1.QuarksJobList{}
	err := client.List(ctx, result, crc.InNamespace(namespace))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list QuarksStatefulSets")
	}

	return result, nil
}
