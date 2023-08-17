package k8s_utils

import (
	"fmt"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	apicoreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

func TaskAddNormalEvent(recorder record.EventRecorder, orderTaskName string) {
	recorder.Eventf(
		&v1alpha1.OrderStep{},
		apicoreV1.EventTypeNormal,
		"Start",
		fmt.Sprintf("New OrderTask %s added to cluster", orderTaskName),
	)
}

func TaskUpgradeEvent(recorder record.EventRecorder, orderTaskName string) {
	recorder.Eventf(
		&v1alpha1.OrderStep{},
		apicoreV1.EventTypeNormal,
		"Upgrade",
		fmt.Sprintf("OrderTask %s upgrade", orderTaskName),
	)
}

func TaskDeleteEvent(recorder record.EventRecorder, orderTaskName string) {
	recorder.Eventf(
		&v1alpha1.OrderStep{},
		apicoreV1.EventTypeWarning,
		"Delete",
		fmt.Sprintf("OrderTask %s deleted", orderTaskName),
	)
}

//
//func TaskErrorEvent(recorder record.EventRecorder, orderTaskName string) {
//	recorder.Eventf(
//		&v1alpha1.OrderStep{},
//		apicoreV1.EventTypeWarning,
//		"Delete",
//		fmt.Sprintf("OrderTask %s deleted", orderTaskName),
//	)
//}
//
//
