package predicate

import (
	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func hasRecreateVolumesAnnotation(obj client.Object) bool {
	_, ok := obj.GetAnnotations()[druidv1alpha1.RecreateVolumesAnnotation]
	return ok
}

// HasRecreateVolumesAnnotation is a predicate for the recreate volumes annotation.
func HasRecreateVolumesAnnotation() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return hasRecreateVolumesAnnotation(event.Object)
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return hasRecreateVolumesAnnotation(event.ObjectNew)
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return hasRecreateVolumesAnnotation(event.Object)
		},
		DeleteFunc: func(_ event.DeleteEvent) bool {
			return true
		},
	}
}
