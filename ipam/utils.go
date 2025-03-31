/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipam

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errNotFound *NotFoundError

// Filter filters a list for a string.
func Filter(list []string, strToFilter string) (newList []string) {
	for _, item := range list {
		if item != strToFilter {
			newList = append(newList, item)
		}
	}
	return
}

// Contains returns true if a list contains a string.
func Contains(list []string, strToSearch string) bool {
	for _, item := range list {
		if item == strToSearch {
			return true
		}
	}
	return false
}

// NotFoundError represents that an object was not found.
type NotFoundError struct {
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return "Object not found"
}

func deepCopyObject(obj client.Object) (client.Object, error) {
	objCopy, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return nil, errors.New("Failed to copy object")
	}
	return objCopy, nil
}

func updateObject(ctx context.Context, cl client.Client, obj client.Object) error {
	objCopy, err := deepCopyObject(obj)
	if err != nil {
		return err
	}
	err = cl.Update(ctx, objCopy)
	if apierrors.IsConflict(err) {
		return WithTransientError(errors.New("Updating object failed"), 0*time.Second)
	}
	return err
}

func createObject(ctx context.Context, cl client.Client, obj client.Object, opts ...client.CreateOption) error {
	objCopy, err := deepCopyObject(obj)
	if err != nil {
		return err
	}
	err = cl.Create(ctx, objCopy, opts...)
	if apierrors.IsAlreadyExists(err) {
		log.Printf("I am inside IsAlreadyExists")
		return WithTransientError(errors.New("Object already exists"), 0*time.Second)
	}
	return err
}

func deleteObject(ctx context.Context, cl client.Client, obj client.Object, opts ...client.DeleteOption) error {
	objCopy, err := deepCopyObject(obj)
	if err != nil {
		return err
	}
	err = cl.Delete(ctx, objCopy, opts...)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// DeleteOwnerRefFromList removes the ownerreference to this Metal3 machine.
func deleteOwnerRefFromList(refList []metav1.OwnerReference,
	objType metav1.TypeMeta, objMeta metav1.ObjectMeta,
) ([]metav1.OwnerReference, error) {
	if len(refList) == 0 {
		return refList, nil
	}
	index, err := findOwnerRefFromList(refList, objType, objMeta)
	if err != nil {
		if ok := errors.As(err, &errNotFound); !ok {
			return nil, err
		}
		return refList, nil
	}
	if len(refList) == 1 {
		return []metav1.OwnerReference{}, nil
	}
	refListLen := len(refList) - 1
	refList[index] = refList[refListLen]
	refList, err = deleteOwnerRefFromList(refList[:refListLen-1], objType, objMeta)
	if err != nil {
		return nil, err
	}
	return refList, nil
}

// SetOwnerRef adds an ownerreference to this Metal3 machine.
func setOwnerRefInList(refList []metav1.OwnerReference, controller bool,
	objType metav1.TypeMeta, objMeta metav1.ObjectMeta,
) ([]metav1.OwnerReference, error) {
	index, err := findOwnerRefFromList(refList, objType, objMeta)
	if err != nil {
		if ok := errors.As(err, &errNotFound); !ok {
			return nil, err
		}
		refList = append(refList, metav1.OwnerReference{
			APIVersion: objType.APIVersion,
			Kind:       objType.Kind,
			Name:       objMeta.Name,
			UID:        objMeta.UID,
			Controller: ptr.To(controller),
		})
	} else {
		// The UID and the APIVersion might change due to move or version upgrade
		refList[index].APIVersion = objType.APIVersion
		refList[index].UID = objMeta.UID
		refList[index].Controller = ptr.To(controller)
	}
	return refList, nil
}

func findOwnerRefFromList(refList []metav1.OwnerReference, objType metav1.TypeMeta,
	objMeta metav1.ObjectMeta,
) (int, error) {
	for i, curOwnerRef := range refList {
		aGV, err := schema.ParseGroupVersion(curOwnerRef.APIVersion)
		if err != nil {
			return 0, err
		}

		bGV, err := schema.ParseGroupVersion(objType.APIVersion)
		if err != nil {
			return 0, err
		}
		// not matching on UID since when pivoting it might change
		// Not matching on API version as this might change
		if curOwnerRef.Name == objMeta.Name &&
			curOwnerRef.Kind == objType.Kind &&
			aGV.Group == bGV.Group {
			return i, nil
		}
	}
	return 0, &NotFoundError{}
}
