// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compaction

import (
	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	druidpredicates "github.com/gardener/etcd-druid/controllers/predicate"

	batchv1 "k8s.io/api/batch/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const controllerName = "compaction-controller"

// RegisterWithManager registers the Compaction Controller with the given controller manager.
func (r *Reconciler) RegisterWithManager(mgr ctrl.Manager) error {
	return ctrl.
		NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.config.Workers,
		}).
		For(&druidv1alpha1.Etcd{}).
		WithEventFilter(predicate.
			Or(druidpredicates.SnapshotRevisionChanged(),
				druidpredicates.JobStatusChanged())).
		Owns(&coordinationv1.Lease{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
