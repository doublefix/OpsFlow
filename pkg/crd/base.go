package crd

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func GetCRDList(crdClient dynamic.ResourceInterface, continueToken string) (*unstructured.UnstructuredList, string, error) {
	crdList, err := crdClient.List(context.TODO(), metav1.ListOptions{
		Limit:    50,
		Continue: continueToken,
	})
	return crdList, crdList.GetContinue(), err
}

func DeleteCRD(crdClient dynamic.ResourceInterface, crdName string) error {
	return crdClient.Delete(context.TODO(), crdName, metav1.DeleteOptions{})
}
