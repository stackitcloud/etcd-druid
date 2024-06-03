package etcd

import (
	"context"
	"time"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	"github.com/gardener/etcd-druid/pkg/utils"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// updateAnnotations removes the operation annotation and sets the recreatedAt annotation, if a volume recreation was
// triggered.
func (r *Reconciler) updateAnnotations(ctx context.Context, logger logr.Logger, etcd *druidv1alpha1.Etcd) error {
	update := false
	patch := client.MergeFrom(etcd.DeepCopy())
	if _, ok := etcd.Annotations[druidv1alpha1.RecreateVolumesAnnotation]; ok {
		update = true
		logger.Info("Removing recreate-volumes annotation", "annotation", druidv1alpha1.RecreateVolumesAnnotation)
		delete(etcd.Annotations, druidv1alpha1.RecreateVolumesAnnotation)
		// we only support volume recreation if the etcd has backups enabled
		if etcd.Spec.Backup.Store != nil && etcd.Spec.Backup.Store.Provider != nil && len(*etcd.Spec.Backup.Store.Provider) > 0 {
			etcd.Annotations[druidv1alpha1.RecreatedAtAnnotation] = time.Now().UTC().Format(time.RFC3339Nano)
		} else {
			r.recorder.Event(etcd, corev1.EventTypeWarning, "SkippingVolumeRecreation", "will not recreate volumes, because backup is not enabled")
		}
	}

	if _, ok := etcd.Annotations[v1beta1constants.GardenerOperation]; ok {
		update = true
		logger.Info("Removing operation annotation", "namespace", etcd.Namespace, "name", etcd.Name, "annotation", v1beta1constants.GardenerOperation)
		delete(etcd.Annotations, v1beta1constants.GardenerOperation)
	}

	if update {
		return r.Patch(ctx, etcd, patch)
	}
	return nil
}

// checkStatefulSetProgress checks if the recreatedAt annotation has already been processed. It recreated-at annotation
// is changed, we should only continue when the STS is healthy.
func (r *Reconciler) checkStatefulSetProgress(ctx context.Context, logger logr.Logger, etcd *druidv1alpha1.Etcd) (requeue bool, err error) {
	recreatedEtcd := etcd.Annotations[druidv1alpha1.RecreatedAtAnnotation]
	if recreatedEtcd == "" {
		return false, nil
	}
	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name:      etcd.Name,
		Namespace: etcd.Namespace,
	}}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	recreatedSTS := sts.Spec.Template.Annotations[druidv1alpha1.RecreatedAtAnnotation]
	if recreatedEtcd == recreatedSTS {
		return false, nil
	}
	if ready, _ := utils.IsStatefulSetReady(etcd.Spec.Replicas, sts); ready {
		return false, nil
	}
	msg := "recreatedAt annotation needs to be propagated, but the Statefulset is not ready yet"
	r.recorder.Event(etcd, corev1.EventTypeWarning, "StatefulSetNotReady", msg)
	logger.Info(msg)
	return true, nil
}
