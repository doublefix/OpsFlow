package crd

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func GetCRDList(crdClient dynamic.ResourceInterface, continueToken string) (*unstructured.UnstructuredList, string, error) {
	crdList, err := crdClient.List(context.TODO(), metav1.ListOptions{
		Limit:    50,
		Continue: continueToken,
	})
	if err != nil {
		return nil, "", err
	}
	if crdList == nil {
		return nil, "", fmt.Errorf("received nil CRD list from API")
	}
	return crdList, crdList.GetContinue(), nil
}

func DeleteCRD(crdClient dynamic.ResourceInterface, crdName string) error {
	return crdClient.Delete(context.TODO(), crdName, metav1.DeleteOptions{})
}
