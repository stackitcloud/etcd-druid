package statefulset

import (
	"context"
	"maps"
	"time"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"

	"github.com/gardener/gardener/pkg/utils/flow"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *component) addTaintPVCsTask(g *flow.Graph, sts *appsv1.StatefulSet, taskIDDependency *flow.TaskID) *flow.TaskID {
	if c.values.RecreatedVolumesAt == "" ||
		sts.Spec.Template.ObjectMeta.Annotations[druidv1alpha1.RecreatedAtAnnotation] == c.values.RecreatedVolumesAt {

		return taskIDDependency
	}

	recreatedTS, err := time.Parse(time.RFC3339Nano, c.values.RecreatedVolumesAt)
	if err != nil {
		c.logger.Error(err, "recreated-at annotation does not contain a valid timestamp, will not recreate volumes")
		// just ignore invalid timestamps, since that means someone messed with the annotations.
		return taskIDDependency
	}

	var (
		dependencies flow.TaskIDs
	)
	if taskIDDependency != nil {
		dependencies = flow.NewTaskIDs(taskIDDependency)
	}

	taskID := g.Add(flow.Task{
		Name: "taint PersistentVolumeClaims",
		Fn: func(ctx context.Context) error {
			return c.deletePVCs(ctx, recreatedTS)
		},
		Dependencies: dependencies,
	})
	c.logger.Info("added taint PersistentVolumeClaims task to the deploy flow", "taskID", taskID, "namespace", c.values.Namespace)

	return &taskID
}

func (c *component) deletePVCs(ctx context.Context, recreatedTS time.Time) error {
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := c.client.List(ctx, pvcList, client.InNamespace(c.values.Namespace), client.MatchingLabels(c.values.Labels)); err != nil {
		return err
	}
	// deletes any PVC that is not already deleting and is older than the "recreatedAt" timestamp
	for _, pvc := range pvcList.Items {
		if !pvc.DeletionTimestamp.IsZero() || pvc.CreationTimestamp.Time.After(recreatedTS) {
			continue
		}
		if err := c.client.Delete(ctx, &pvc); client.IgnoreNotFound(err) != nil {
			return err
		}

		c.logger.Info("deleted old PersistentVolumeClaim", "namespace", c.values.Namespace, "name", c.values.Name, "pvc", pvc.Name)
	}
	return nil
}

// getPodAnnotations sets the annotations to val.Annotations, preserving any existing RecreatedAtAnnotation unless
// val.RecreatedAt is set.
func getPodAnnotations(val Values, sts *appsv1.StatefulSet) map[string]string {
	res := make(map[string]string, len(val.Annotations))
	maps.Copy(res, val.Annotations)
	if recreateTS, ok := sts.Spec.Template.ObjectMeta.Annotations[druidv1alpha1.RecreatedAtAnnotation]; ok {
		res[druidv1alpha1.RecreatedAtAnnotation] = recreateTS
	}
	if val.RecreatedVolumesAt != "" {
		res[druidv1alpha1.RecreatedAtAnnotation] = val.RecreatedVolumesAt
	}

	return res
}
