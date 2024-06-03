package statefulset

import (
	"context"
	"time"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("getPodAnnotations", func() {
	var (
		sts *appsv1.StatefulSet
	)
	BeforeEach(func() {
		sts = &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
		}
	})
	It("should keep the annotation", func() {
		sts.Spec.Template.Annotations[druidv1alpha1.RecreatedAtAnnotation] = "foo"
		res := getPodAnnotations(Values{}, sts)
		Expect(res).To(Equal(map[string]string{
			druidv1alpha1.RecreatedAtAnnotation: "foo",
		}))
	})
	It("should overwrite the annotation", func() {
		val := Values{
			RecreatedVolumesAt: "bar",
		}
		res := getPodAnnotations(val, sts)
		Expect(res).To(Equal(map[string]string{
			druidv1alpha1.RecreatedAtAnnotation: "bar",
		}))
	})
	It("should add the value annotations", func() {
		sts.Spec.Template.Annotations[druidv1alpha1.RecreatedAtAnnotation] = "foo"
		val := Values{
			Annotations: map[string]string{
				"key": "value",
			},
		}
		res := getPodAnnotations(val, sts)
		Expect(res).To(Equal(map[string]string{
			druidv1alpha1.RecreatedAtAnnotation: "foo",
			"key":                               "value",
		}))
	})
})

var _ = Describe("deletePVCs", func() {
	var (
		kube client.Client
		val  Values
		ctx  = context.Background()
		c    *component
		now  time.Time
	)
	BeforeEach(func() {
		val = Values{
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		}
		now = time.Now()

		kube = fake.NewClientBuilder().
			WithObjects(
				createPVC("pvc1", val, now),
				createPVC("pvc2", val, now),
			).Build()
		c = &component{
			values: val,
			client: kube,
			logger: GinkgoLogr,
		}
	})
	It("should delete older pvcs", func() {
		recreatedAt := now.Add(1 * time.Minute)
		Expect(c.deletePVCs(ctx, recreatedAt)).To(Succeed())
		pvcList := &corev1.PersistentVolumeClaimList{}
		Expect(kube.List(ctx, pvcList, client.InNamespace(val.Namespace))).To(Succeed())
		Expect(pvcList.Items).To(BeEmpty())
	})
	It("should keep newer pvcs", func() {
		recreatedAt := now.Add(-1 * time.Minute)
		Expect(c.deletePVCs(ctx, recreatedAt)).To(Succeed())
		pvcList := &corev1.PersistentVolumeClaimList{}
		Expect(kube.List(ctx, pvcList, client.InNamespace(val.Namespace))).To(Succeed())
		Expect(pvcList.Items).To(HaveLen(2))
	})
})

func createPVC(name string, val Values, t time.Time) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         val.Namespace,
			CreationTimestamp: metav1.NewTime(t),
			Labels:            val.Labels,
		},
	}
}
