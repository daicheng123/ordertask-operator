package main

import (
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	"github.com/daicheng123/ordertask-operator/cmd/ordertask/utils"
	"github.com/daicheng123/ordertask-operator/controllers/order_task"
	"log"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	Leader_Election_ID = "order-task-operator"
)

type Operator struct {
	*utils.OperatorFlags
	StopChan chan struct{}
}

func NewOperator() *Operator {
	flags := new(utils.OperatorFlags)
	flags.Init()
	return &Operator{
		OperatorFlags: flags,
		StopChan:      make(chan struct{}),
	}
}

func (o *Operator) Run() error {
	kc, err := utils.LoadKubernetesConfig(o.OperatorFlags)
	if err != nil {
		return err
	}

	logf.SetLogger(zap.New())
	mgr, err := manager.New(kc, manager.Options{
		Logger:             logf.Log.WithName(v1alpha1.OrderTaskResourceKind),
		LeaderElection:     o.EnableLeaderElection,
		LeaderElectionID:   Leader_Election_ID,
		MetricsBindAddress: o.MetricsAddr,
		Cache: cache.Options{
			Namespaces: []string{utils.GetNamespace()},
		},
	})

	if err != nil {
		log.Printf("failed to set up manager, error: %s.", err.Error())
		return err
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(mgr.GetScheme())
	if err != nil {
		mgr.GetLogger().Error(err, "failed to add schema.")
		return err
	}
	//metrics.Registry
	go func() {
		http.ListenAndServe(o.ListenAddr, nil)
	}()

	_, crdCli, apiextCli, err := utils.CreateOperatorClients(o.OperatorFlags)
	if err != nil {
		mgr.GetLogger().Error(err, "failed to create client sets.")
		return err
	}
	reconciler, err := order_task.NewReconciler(mgr, crdCli, apiextCli)
	if err != nil {
		mgr.GetLogger().Error(err, "failed to create reconciler.")
		return err
	}
	if err = ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OrderStep{}).
		Complete(reconciler); err != nil {
		mgr.GetLogger().Error(err, "failed to set up order task controller.")
		return err
	}

	err = mgr.Start(signals.SetupSignalHandler())
	if err != nil {
		mgr.GetLogger().Error(err, "unable to start manager.")
	}
	return err
}

func main() {

}
