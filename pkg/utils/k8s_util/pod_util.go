package k8s_util

import (
	"context"
	"fmt"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	"github.com/daicheng123/ordertask-operator/pkg/utils/list"
	"github.com/daicheng123/ordertask-operator/pkg/utils/retry_util"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func RetryCreateAndWaitPod(ctx context.Context, cli client.Client, pod *corev1.Pod, interval time.Duration, maxRetries int) (*corev1.Pod, error) {
	var err error
	if err = cli.Create(ctx, pod); err != nil {
		return nil, err
	}
	var retPod *corev1.Pod
	err = retry_util.Retry(interval, maxRetries, func() (bool, error) {
		err := cli.Get(ctx, client.ObjectKeyFromObject(pod), retPod)
		if err != nil {
			return false, err
		}
		switch retPod.Status.Phase {
		case corev1.PodRunning:
			return true, nil
		case corev1.PodPending:
			return false, nil
		default:
			return false, fmt.Errorf("unexpected pod status.phase: %v", retPod.Status.Phase)
		}
	})

	if err != nil {
		if retry_util.IsRetryFailure(err) {
			return nil, fmt.Errorf("failed to wait pod running, it is still pending: %v", err)
		}
		return nil, fmt.Errorf("failed to wait pod running: %v", err)
	}
	return retPod, nil
}

func RetryPushPod2List(_ context.Context, sl *list.SafeListLimited, task *v1alpha1.OrderStep, interval time.Duration, maxRetries int) error {
	var err error
	err = retry_util.Retry(interval, maxRetries, func() (bool, error) {
		if !sl.PushFront(task) {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		if retry_util.IsRetryFailure(err) {
			return fmt.Errorf("event_push_queue: queue is full, event: %v", err)
		}
		return fmt.Errorf("event_push_queue: failed to push pod into queue: %v", err)
	}
	return nil
}
