package pod_builder

import (
	"context"
	"fmt"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	image2 "github.com/daicheng123/ordertask-operator/pkg/image"
	"github.com/daicheng123/ordertask-operator/pkg/utils/k8s_util"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/lru"
	"runtime"
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
	orderTaskNamePrefix          = "order-task-"
	initContainerPath            = "chengdai/entrypoint"
	annotationsOrderField        = "orderField"
	annotationsOrderInitialValue = "0"

	EntryPointVolume    = "entrypoint-volume"
	DevopsScriptsVolume = "scripts-volume"
	PodInfoVolume       = "podinfo"
)

var (
	osArch = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
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
	if len(step.Command) == 0 {
		imageInfo, err := pb.getImageInfoWithName(step.Image)
		if err != nil {
			return step.Container
		}

		if imageCmd, ok := imageInfo.Command[osArch]; ok {
			step.Command = imageCmd.Command
			if len(step.Args) == 0 {
				step.Args = imageCmd.Args
			}
		} else {
			return step.Container // error image command
		}
		//step.Command = imageInfo.Command[osArch].Command
	}

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
		annotationsOrderField: annotationsOrderInitialValue,
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
	_, err := k8s_util.RetryCreateAndWaitPod(ctx, pb.Client, pb.pod, time.Second, 3)

	return err
}

func NewPodBuilder(task *v1alpha1.OrderStep, client client.Client, cache *lru.Cache) *PodBuilder {
	return &PodBuilder{
		task:       task,
		Client:     client,
		imageCache: cache,
	}
}

func (pb *PodBuilder) getImageInfoWithName(imageName string) (*image2.ImageInfo, error) {
	ref, err := name.ParseReference(imageName, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	var imageInfo *image2.ImageInfo
	if v, ok := pb.imageCache.Get(ref); ok {
		imageInfo = v.(*image2.ImageInfo)
	} else {
		imageInfo, err := image2.ParseImage(imageName)
		if err != nil {
			return nil, err
		}
		pb.imageCache.Add(ref, imageInfo)
	}
	return imageInfo, nil
}

func generateBaseName(name string) string {
	taskName := orderTaskNamePrefix + strings.ReplaceAll(name, "_", "-")
	return strings.ToLower(taskName)
}
