package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type Pods interface {
	GetPod(namespace string, name string) (*corev1.Pod, error)
	CreatePod(namespace string, pod *corev1.Pod) error
	UpdatePod(namespace string, pod *corev1.Pod) error
	CreateOrUpdatePod(namespace string, pod *corev1.Pod) error
	DeletePod(namespace string, name string) error
	ListPods(namespace string) (*corev1.PodList, error)
	UpdatePodLabels(namespace, podName string, labels map[string]string) error
}

type PodsService struct {
	kubeClient rest.Interface
	//logger     log.Logger
	//metricsRecorder metrics.Recorder
}
