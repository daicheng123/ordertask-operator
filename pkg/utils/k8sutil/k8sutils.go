package k8sutil

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func IsKubernetesResourceAlreadyExistError(err error) bool {
	return apierrors.IsAlreadyExists(err)
}
