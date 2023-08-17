package pod_manager

import (
	"context"
	"fmt"
	"github.com/daicheng123/ordertask-operator/api/tasks/v1alpha1"
	image2 "github.com/daicheng123/ordertask-operator/pkg/image"
	"github.com/daicheng123/ordertask-operator/pkg/utils/k8s_utils"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/lru"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
)

type PodManagerInterface interface {
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
	annotationTaskExistValue     = "-1"

	EntryPointVolume    = "entrypoint-volume"
	DevopsScriptsVolume = "scripts-volume"
	PodInfoVolume       = "podinfo"
)

var (
	osArch = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

type PodManager struct {
	pod        *corev1.Pod
	task       *v1alpha1.OrderStep
	Client     client.Client
	imageCache *lru.Cache
}

func (pm *PodManager) setInitContainer() {
	initContainer := corev1.Container{
		Name:    generateBaseName(pm.task.GetName()) + "-init",
		Image:   initContainerPath,
		Command: []string{"cp", "/app/entrypoint", "/entrypoint/bin/"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "entrypoint-volume",
				MountPath: "/entrypoint/bin/",
			},
		},
	}
	pm.pod.Spec.InitContainers = []corev1.Container{
		initContainer,
	}
}

func (pm *PodManager) setContainer(index int, step v1alpha1.Step) corev1.Container {
	if len(step.Command) == 0 {
		imageInfo, err := pm.getImageInfoWithName(step.Image)
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

func (pm *PodManager) setPodVolumes() {
	pm.pod.Spec.Volumes = []corev1.Volume{
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

func (pm *PodManager) setPodMeta() {
	pm.pod.SetName(generateBaseName(pm.task.GetName()))
	pm.pod.SetNamespace(pm.task.GetNamespace())

	pm.pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	annotations := map[string]string{
		annotationsOrderField: annotationsOrderInitialValue,
	}
	pm.pod.SetAnnotations(annotations)
}

func (pm *PodManager) Builder(ctx context.Context) error {
	pod, err := pm.getChildPod(ctx)
	if err == nil {
		if pod.Status.Phase == corev1.PodRunning && pod.GetAnnotations()[annotationsOrderField] == annotationsOrderField {
			pod.GetAnnotations()[annotationsOrderField] = "1"
			return pm.Client.Update(ctx, pod)
		} else {
			if err = pm.forward(ctx, pod); err != nil {
				return err
			}
		}
		return nil
	}

	pm.pod = new(corev1.Pod)
	pm.setPodMeta()
	pm.setInitContainer()

	containers := make([]corev1.Container, 0, len(pm.task.Spec.Steps))

	for i := 0; i < len(pm.task.Spec.Steps); i++ {
		containers = append(containers, pm.setContainer(i+1, pm.task.Spec.Steps[i]))
	}
	pm.pod.Spec.Containers = containers
	pm.setPodVolumes()

	// set owner
	pm.pod.OwnerReferences = append(pm.pod.OwnerReferences, metav1.OwnerReference{
		APIVersion: pm.pod.APIVersion,
		Kind:       pm.pod.Kind,
		Name:       pm.pod.Name,
		UID:        pm.pod.UID,
	})
	_, err = k8s_utils.RetryCreateAndWaitPod(ctx, pm.Client, pm.pod, time.Second, 3)
	return err
}

func NewPodManager(task *v1alpha1.OrderStep, client client.Client, cache *lru.Cache) *PodManager {
	return &PodManager{
		task:       task,
		Client:     client,
		imageCache: cache,
	}
}

func (pm *PodManager) getImageInfoWithName(imageName string) (*image2.ImageInfo, error) {
	ref, err := name.ParseReference(imageName, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	var imageInfo *image2.ImageInfo
	if v, ok := pm.imageCache.Get(ref); ok {
		imageInfo = v.(*image2.ImageInfo)
	} else {
		imageInfo, err := image2.ParseImage(imageName)
		if err != nil {
			return nil, err
		}
		pm.imageCache.Add(ref, imageInfo)
	}
	return imageInfo, nil
}

func (pm *PodManager) getChildPod(ctx context.Context) (*corev1.Pod, error) {

	pod := &corev1.Pod{}
	err := pm.Client.Get(ctx, types.NamespacedName{
		Namespace: pm.task.Namespace,
		Name:      orderTaskNamePrefix + pm.task.Name}, pod)

	if err != nil {
		return nil, err
	}
	return pod, err
}

func (pm *PodManager) forward(ctx context.Context, pod *corev1.Pod) error {
	if pod.Status.Phase == corev1.PodSucceeded {
		return nil
	}
	if pod.Annotations[annotationsOrderField] == annotationTaskExistValue {
		return nil
	}
	order, err := strconv.Atoi(pod.Annotations[annotationsOrderField])
	if err != nil {
		return nil
	}
	if order == len(pod.Spec.Containers) {
		return nil
	}

	if pod.Status.ContainerStatuses[order-1].State.Terminated == nil {
		return nil
	} else {
		if pod.Status.ContainerStatuses[order-1].State.Terminated.ExitCode != 0 {
			pod.Annotations[annotationsOrderField] = annotationTaskExistValue
			return pm.Client.Update(ctx, pod)
		}
	}
	order++
	pod.Annotations[annotationsOrderField] = strconv.Itoa(order)
	return pm.Client.Update(ctx, pod)
}

func generateBaseName(name string) string {
	taskName := orderTaskNamePrefix + strings.ReplaceAll(name, "_", "-")
	return strings.ToLower(taskName)
}
