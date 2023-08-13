package order_task

import (
	"context"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	"github.com/daicheng123/ordertask-operator/builders/pod_builder"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/lru"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type OrderTaskController struct {
	client.Client
	Event record.EventRecorder // record event
	*clientset.Clientset
	imageCache *lru.Cache
}

func (otc *OrderTaskController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	ot := &v1alpha1.OrderStep{}
	err := otc.Get(ctx, req.NamespacedName, ot)
	if err != nil {
		return reconcile.Result{}, err
	}

	podBuilder := pod_builder.NewPodBuilder(ot, otc.Client, otc.imageCache)
	if err = podBuilder.Builder(ctx); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (otc *OrderTaskController) InjectClient(client client.Client) error {
	otc.Client = client
	return nil
}
