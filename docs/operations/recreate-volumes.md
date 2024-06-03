# Recreating volumes

In just our fork, we added the ability to recreate volumes, as a one-off-operation. The recreation is triggered by
annotating the etcd CR with `druid.gardener.cloud/recreate-volumes=true`. This triggers the following sequence of
events:

1. The annotation is removed, and a new annotation `druid.gardener.cloud/recreated-at` is added with the current timestamp.
2. All PVCs which match the `.spec.labels` and which are older than this timestamp are deleted
    - they will not be actually deleted until the Pod using them restarts
3. The annotation `druid.gardener.cloud/recreated-at` is added to the PodTemplate annotations, triggering a new rollout of the StatefulSet
4. The pods are recreated on-by-one by the statefulset-controller

Unfortunately, etcd-druid uses `podManagementPolicy: Parallel` for the StatefulSet, so if any other change is applied
during the recreation, etcd could be in trouble. There are some safeguards in place, but it is best to ensure there are
no further updates during that time.

## Script for recreating etcd-main and blocking changes at the same time

In our environment, we can block customer changes by setting a `stackit.cloud/readonly` annotation on the shoot:

```bash
#!/usr/bin/env bash

# usage: ./recreate.sh prd project shoot

set -euo pipefail

ENV="$1"
project="$2"
shoot="$3"

gardenctl target --garden "$ENV"
projectNS=$(kubectl get project "$project" -o jsonpath="{.spec.namespace}")

kubectl annotate shoot -n "$projectNS" "$shoot" stackit.cloud/readonly=true
kubectl annotate shoot -n "$projectNS" "$shoot" "stackit.cloud/readonly-message=System Maintenance in Progress"

gardenctl target --garden "$ENV" --project "$project" --shoot "$shoot" --control-plane

kubectl annotate etcd etcd-main druid.gardener.cloud/recreate-volumes=true

echo "> Waiting for STS to be updated"
kubectl wait --for=jsonpath='{.status.updatedReplicas}'=1 sts etcd-main --timeout=1m

echo "> Waiting for STS to be rolled out"
kubectl wait --for=jsonpath='{.status.updatedReplicas}'=3 sts etcd-main --timeout=10m

echo "> Waiting for STS to be ready"
kubectl wait --for=jsonpath='{.status.readyReplicas}'=3 sts etcd-main --timeout=2m

gardenctl target --garden "$ENV"
kubectl annotate shoot -n "$projectNS" "$shoot" stackit.cloud/readonly-
kubectl annotate shoot -n "$projectNS" "$shoot" stackit.cloud/readonly-message-

```
