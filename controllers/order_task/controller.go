package order_task

import (
	"context"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	"github.com/daicheng123/ordertask-operator/builders/pod_builder"
	"github.com/daicheng123/ordertask-operator/pkg/k8s/clientset/versioned"
	"github.com/daicheng123/ordertask-operator/pkg/utils/k8s_util"
	"github.com/daicheng123/ordertask-operator/pkg/utils/list"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sErr "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/lru"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	// NotifyConcurrency = 20
	defaultImageSize     = 100
	defaultEvictPoolSize = 100
)

type OrderTaskReconciler interface {
	reconcile.Reconciler
	OnUpdateFunc(context.Context, event.UpdateEvent, workqueue.RateLimitingInterface)
}

type OrderTaskController struct {
	crdCli     *versioned.Clientset
	manager    manager.Manager
	imageCache *lru.Cache
	eventQueue *list.SafeListLimited
	errorChan  chan error
}

func NewReconciler(mgr manager.Manager, crdCli *versioned.Clientset, apiextCli *apiextensionsclient.Clientset) (OrderTaskReconciler, error) {
	reconciler := &OrderTaskController{
		manager: mgr,
		crdCli:  crdCli,
		imageCache: lru.NewWithEvictionFunc(defaultImageSize, func(key lru.Key, value interface{}) {

		}),

		//eventQueue: list.NewSafeListLimited(1000),
		//errorChan:  make(chan error),
	}
	//go reconciler.processTaskEventsQueue()

	return reconciler, reconciler.createCustomResourceDefinition(context.Background(), apiextCli)
}

func (otc *OrderTaskController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	ot := &v1alpha1.OrderStep{}
	client := otc.manager.GetClient()
	err := client.Get(ctx, req.NamespacedName, ot)

	if err == nil || (err != nil && k8s_util.IsKubernetesResourceNotExist(err)) {

		podBuilder := pod_builder.NewPodBuilder(ot, client, otc.imageCache)
		if err = podBuilder.Builder(ctx); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, err
}

func (otc *OrderTaskController) createCustomResourceDefinition(ctx context.Context, apiextCli *apiextensionsclient.Clientset) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: v1alpha1.OrderTaskCRDName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group: api.SchemeGroupVersion.Group,
			Versions: []apiextensionsv1beta1.CustomResourceDefinitionVersion{
				{
					Name:    v1alpha1.OrderTaskVersion,
					Storage: true,
					Served:  true,
				},
			},
			Scope: apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: v1alpha1.OrderTaskResourcePlural,
				Kind:   reflect.TypeOf(v1alpha1.OrderStep{}).Name(),
			},
		},
	}
	_, err := apiextCli.ApiextensionsV1beta1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
	if err != nil && !k8s_util.IsKubernetesResourceAlreadyExistError(err) {
		return err
	}
	// wait for order task crd resource being created
	otc.manager.GetLogger().Info("creating crd resource, wating till its established")
	err = wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 60*time.Second, false, func(ctx context.Context) (done bool, err error) {
		crd, err = apiextCli.ApiextensionsV1beta1().CustomResourceDefinitions().Get(ctx, v1alpha1.OrderTaskCRDName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					//otc.logger.WithName().
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := apiextCli.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(ctx, v1alpha1.OrderTaskCRDName, metav1.DeleteOptions{})
		if deleteErr != nil {
			return k8sErr.NewAggregate([]error{err, deleteErr})
		}
		return err
	}
	return nil
}

func (otc *OrderTaskController) OnUpdateFunc(_ context.Context, event event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
	for _, ref := range event.ObjectNew.GetOwnerReferences() {
		if ref.Kind == v1alpha1.OrderTaskResourceKind && ref.APIVersion == v1alpha1.OrderTaskApiVersionGroup {
			limitingInterface.Add(reconcile.Request{
				types.NamespacedName{
					Name: ref.Name, Namespace: event.ObjectNew.GetNamespace(),
				},
			})
		}
	}
}

//
//func (otc *OrderTaskController) processTaskEventsQueue(stopCh <-chan struct{}, wg *sync.WaitGroup) {
//	sema := semaphore.NewSemaphore(NotifyConcurrency)
//	duration := time.Duration(100) * time.Millisecond
//	for {
//		events := otc.EventQueue.PopBackBy(100)
//		if len(events) == 0 {
//			time.Sleep(duration)
//			continue
//		}
//		otc.process(events, sema)
//	}
//}
//
//func (otc *OrderTaskController) process(events []interface{}, sema *semaphore.Semaphore) {
//	for i := range events {
//		if events[i] == nil {
//			continue
//		}
//
//		event := events[i].(*v1alpha1.OrderStep)
//		sema.Acquire()
//		go func(event *v1alpha1.OrderStep) {
//			defer sema.Release()
//			//e.consumeOne(event)
//		}(event)
//	}

//}
