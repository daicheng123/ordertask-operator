package pod_builder

import (
	"context"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	"github.com/daicheng123/ordertask-operator/pkg/utils/k8s_util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/lru"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
)

type PodBuilderInterface interface {
	setInitContainer()
	setContainer(index int, step v1alpha1.OrderStep) (corev1.Container, error)
	setPodVolumes()
	setPodMeta()
	Builder(ctx context.Context) *corev1.Pod
}

const (
	orderTaskNamePrefix = "order-task-"
	orderField          = "order"
	initContainerPath   = "chengdai/entrypoint"

	EntryPointVolume    = "entrypoint-volume"
	DevopsScriptsVolume = "scripts-volume"
	PodInfoVolume       = "podinfo"
)

type PodBuilder struct {
	pod        *corev1.Pod
	task       *v1alpha1.OrderStep
	Client     client.Client
	imageCache *lru.Cache
}

func (pb *PodBuilder) setInitContainer() {
	initContainer := corev1.Container{
		Name:    generateBaseName(pb.task.GetName()) + "-init",
		Image:   initContainerPath,
		Command: []string{"cp", "/app/entrypoint", "/entrypoint/bin/"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "entrypoint-volume",
				MountPath: "/entrypoint/bin/",
			},
		},
	}
	pb.pod.Spec.InitContainers = []corev1.Container{
		initContainer,
	}
}

func (pb *PodBuilder) setContainer(index int, step v1alpha1.Step) corev1.Container {

	container := corev1.Container{
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/entrypoint/bin/entrypoint"},
		Args: []string{
			"--wait", "/etc/podinfo/order",
			"--waitcontent", strconv.Itoa(index + 1),
			"--out", "stdout",
			"--command",
		},
	}
	// shc -c
	container.Args = append(container.Args, strings.Join(step.Command, " "))
	container.Args = append(container.Args, step.Args...)

	container.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "entrypoint-volume",
			MountPath: "/entrypoint/bin/",
		},
		{
			Name:      "podinfo",
			MountPath: "/etc/podinfo",
		},
	}

	return container
}

func (pb *PodBuilder) setPodVolumes() {
	pb.pod.Spec.Volumes = []corev1.Volume{
		{
			Name: EntryPointVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: DevopsScriptsVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: PodInfoVolume,
			VolumeSource: corev1.VolumeSource{
				DownwardAPI: &corev1.DownwardAPIVolumeSource{
					Items: []corev1.DownwardAPIVolumeFile{
						{
							Path: "order",
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: `metadata.annotations['taskorder']`,
							},
						},
					},
				},
			},
		},
	}
}

func (pb *PodBuilder) setPodMeta() {
	pb.pod.SetName(generateBaseName(pb.task.GetName()))
	pb.pod.SetNamespace(pb.task.GetNamespace())

	pb.pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	annotations := map[string]string{
		orderField: "0",
	}
	pb.pod.SetAnnotations(annotations)
}

func (pb *PodBuilder) Builder(ctx context.Context) error {
	pb.pod = new(corev1.Pod)
	pb.setPodMeta()
	pb.setInitContainer()

	containers := make([]corev1.Container, 0, len(pb.task.Spec.Steps))

	for i := 0; i < len(pb.task.Spec.Steps); i++ {
		containers = append(containers, pb.setContainer(i+1, pb.task.Spec.Steps[i]))
	}
	pb.pod.Spec.Containers = containers
	pb.setPodVolumes()

	// set owner
	pb.pod.OwnerReferences = append(pb.pod.OwnerReferences, metav1.OwnerReference{
		APIVersion: pb.pod.APIVersion,
		Kind:       pb.pod.Kind,
		Name:       pb.pod.Name,
		UID:        pb.pod.UID,
	})
	_, err := k8s_util.CreateAndWaitPod(ctx, pb.Client, pb.pod, 3*time.Second, 3)

	return err
}

func NewPodBuilder(task *v1alpha1.OrderStep, client client.Client, cache *lru.Cache) *PodBuilder {
	return &PodBuilder{
		task:       task,
		Client:     client,
		imageCache: cache,
	}
}

func generateBaseName(name string) string {
	taskName := orderTaskNamePrefix + strings.ReplaceAll(name, "_", "-")
	return strings.ToLower(taskName)
}
