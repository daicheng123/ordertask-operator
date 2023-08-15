package k8s_util

import (
	"context"
	"fmt"
	"github.com/daicheng123/ordertask-operator/pkg/utils/retry_util"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func CreateAndWaitPod(ctx context.Context, cli client.Client, pod *corev1.Pod, interval time.Duration, maxRetries int) (*corev1.Pod, error) {
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
