package order_task

import (
	"context"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	"github.com/daicheng123/ordertask-operator/builders/pod_builder"
	"github.com/daicheng123/ordertask-operator/pkg/k8s/clientset/versioned"
	"github.com/daicheng123/ordertask-operator/pkg/utils/k8sutil"
	"github.com/go-logr/logr"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sErr "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/lru"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type OrderTaskController struct {
	client.Client
	crdCli     *versioned.Clientset
	apiextCli  *apiextensionsclient.Clientset
	Event      record.EventRecorder // record event
	logger     logr.Logger
	scheme     *runtime.Scheme
	imageCache *lru.Cache
}

func NewReconciler(mgr manager.Manager, crdCli *versioned.Clientset, apiextCli *apiextensionsclient.Clientset) (reconcile.Reconciler, error) {
	reconciler := &OrderTaskController{
		Client: mgr.GetClient(), Event: mgr.GetEventRecorderFor(v1alpha1.OrderTaskResourceKind),
		crdCli: crdCli, scheme: mgr.GetScheme(),
		logger: mgr.GetLogger(), apiextCli: apiextCli,
	}
	return reconciler, reconciler.createCustomResourceDefinition(context.Background())
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

func (otc *OrderTaskController) createCustomResourceDefinition(ctx context.Context) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: v1alpha1.OrderTaskCRDName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   api.SchemeGroupVersion.Group,
			Version: v1alpha1.OrderTaskVersion,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: v1alpha1.OrderTaskResourcePlural,
				Kind:   reflect.TypeOf(v1alpha1.OrderStep{}).Name(),
			},
		},
	}
	_, err := otc.apiextCli.ApiextensionsV1beta1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
	if err != nil && !k8sutil.IsKubernetesResourceAlreadyExistError(err) {
		return err
	}
	// wait for order task crd resource being created
	otc.logger.Info("creating crd resource, wating till its established")
	err = wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 60*time.Second, false, func(ctx context.Context) (done bool, err error) {
		crd, err = otc.apiextCli.ApiextensionsV1beta1().CustomResourceDefinitions().Get(ctx, v1alpha1.OrderTaskCRDName, metav1.GetOptions{})
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
		deleteErr := otc.apiextCli.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(ctx, v1alpha1.OrderTaskCRDName, metav1.DeleteOptions{})
		if deleteErr != nil {
			return k8sErr.NewAggregate([]error{err, deleteErr})
		}
		return err
	}
	return nil
}
