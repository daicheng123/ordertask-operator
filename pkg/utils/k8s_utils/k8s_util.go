package k8s_utils

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func IsKubernetesResourceAlreadyExistError(err error) bool {
	return apierrors.IsAlreadyExists(err)
}

func IsKubernetesResourceNotExist(err error) bool {
	return apierrors.IsNotFound(err)
}
