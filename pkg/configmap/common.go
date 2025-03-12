package configmap

import (
	"github.com/modcoco/OpsFlow/pkg/core"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteConfigMapsByLabel(appCtx core.AppContext, namespace, labelSelector string) error {
	configMaps, err := appCtx.Client().Core().CoreV1().ConfigMaps(namespace).List(appCtx.Ctx(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	for _, cm := range configMaps.Items {
		err := appCtx.Client().Core().CoreV1().ConfigMaps(namespace).Delete(appCtx.Ctx(), cm.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
