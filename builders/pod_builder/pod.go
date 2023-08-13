package pod_builder

import (
	"context"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	"k8s.io/utils/lru"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodBuilder struct {
	task       *v1alpha1.OrderStep
	Client     client.Client
	imageCache *lru.Cache
}

func NewPodBuilder(task *v1alpha1.OrderStep, client client.Client, cache *lru.Cache) *PodBuilder {
	return &PodBuilder{
		task:       task,
		Client:     client,
		imageCache: cache,
	}
}

func (pb *PodBuilder) Builder(ctx context.Context) error {

	return nil
}
