package svc

import (
	"github.com/modcoco/OpsFlow/pkg/core"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteServicesByLabel(appCtx core.AppContext, namespace, labelSelector string) error {
	services, err := appCtx.Client().Core().CoreV1().Services(namespace).List(appCtx.Ctx(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	for _, svc := range services.Items {
		err := appCtx.Client().Core().CoreV1().Services(namespace).Delete(appCtx.Ctx(), svc.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
