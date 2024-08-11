// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package otelcolresources

import (
	"context"
	"errors"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/dash0hq/dash0-operator/internal/dash0/util"
)

type OTelColResourceManager struct {
	client.Client
	OTelCollectorNamePrefix string
}

func (m *OTelColResourceManager) CreateOrUpdateOpenTelemetryCollectorResources(
	ctx context.Context,
	namespace string,
	images util.Images,
	ingressEndpoint string,
	authorizationToken string,
	secretRef string,
	logger *logr.Logger,
) (bool, bool, error) {
	config := &oTelColConfig{
		Namespace:          namespace,
		NamePrefix:         m.OTelCollectorNamePrefix,
		IngressEndpoint:    ingressEndpoint,
		SecretRef:          secretRef,
		Images:             images,
		AuthorizationToken: authorizationToken,
	}
	desiredState, err := assembleDesiredState(config)
	if err != nil {
		return false, false, err
	}
	resourcesHaveBeenCreated := false
	resourcesHaveBeenUpdated := false
	for _, desiredResource := range desiredState {
		isNew, isChanged, err := m.createOrUpdateResource(
			ctx,
			desiredResource,
			logger,
		)
		if err != nil {
			return resourcesHaveBeenCreated, resourcesHaveBeenUpdated, err
		} else if isNew {
			resourcesHaveBeenCreated = true
		} else if isChanged {
			resourcesHaveBeenUpdated = true
		}
	}

	return resourcesHaveBeenCreated, resourcesHaveBeenUpdated, nil
}

func (m *OTelColResourceManager) createOrUpdateResource(
	ctx context.Context,
	desiredObject client.Object,
	logger *logr.Logger,
) (bool, bool, error) {
	existingObject, err := m.createEmptyReceiverFor(desiredObject)
	if err != nil {
		return false, false, err
	}
	err = m.Client.Get(ctx, client.ObjectKeyFromObject(desiredObject), existingObject)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, false, err
		}
		err = m.createResource(ctx, desiredObject, logger)
		if err != nil {
			return false, false, err
		}
		return true, false, nil
	} else {
		// object needs to be updated
		hasChanged, err := m.updateResource(ctx, existingObject, desiredObject, logger)
		if err != nil {
			return false, false, err
		}
		return false, hasChanged, err
	}
}

func (m *OTelColResourceManager) createEmptyReceiverFor(desiredObject client.Object) (client.Object, error) {
	objectKind := desiredObject.GetObjectKind()
	gvk := schema.GroupVersionKind{
		Group:   objectKind.GroupVersionKind().Group,
		Version: objectKind.GroupVersionKind().Version,
		Kind:    objectKind.GroupVersionKind().Kind,
	}
	runtimeObject, err := scheme.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	return runtimeObject.(client.Object), nil
}

func (m *OTelColResourceManager) createResource(
	ctx context.Context,
	desiredObject client.Object,
	logger *logr.Logger,
) error {
	err := m.Client.Create(ctx, desiredObject)
	if err != nil {
		return err
	}
	logger.Info(
		"created resource",
		"name",
		desiredObject.GetName(),
		"namespace",
		desiredObject.GetNamespace(),
		"kind",
		desiredObject.GetObjectKind().GroupVersionKind(),
	)
	return nil
}

func (m *OTelColResourceManager) updateResource(
	ctx context.Context,
	existingObject client.Object,
	desiredObject client.Object,
	logger *logr.Logger,
) (bool, error) {
	logger.Info(
		"updating resource",
		"name",
		desiredObject.GetName(),
		"namespace",
		desiredObject.GetNamespace(),
		"kind",
		desiredObject.GetObjectKind().GroupVersionKind(),
	)
	err := m.Client.Update(ctx, desiredObject)
	if err != nil {
		return false, err
	}
	updatedObject, err := m.createEmptyReceiverFor(desiredObject)
	if err != nil {
		return false, err
	}
	err = m.Client.Get(ctx, client.ObjectKeyFromObject(desiredObject), updatedObject)
	if err != nil {
		return false, err
	}
	hasChanged := !reflect.DeepEqual(existingObject, updatedObject)
	if hasChanged {
		logger.Info(
			"updated resource",
			"name",
			desiredObject.GetName(),
			"namespace",
			desiredObject.GetNamespace(),
			"kind",
			desiredObject.GetObjectKind().GroupVersionKind(),
			"diff",
			cmp.Diff(existingObject, updatedObject),
		)
	}
	return hasChanged, nil
}

func (m *OTelColResourceManager) DeleteResources(
	ctx context.Context,
	namespace string,
	images util.Images,
	ingressEndpoint string,
	authorizationToken string,
	secretRef string,
	logger *logr.Logger,
) error {
	config := &oTelColConfig{
		Namespace:          namespace,
		NamePrefix:         m.OTelCollectorNamePrefix,
		IngressEndpoint:    ingressEndpoint,
		SecretRef:          secretRef,
		Images:             images,
		AuthorizationToken: authorizationToken,
	}
	allObjects, err := assembleDesiredState(config)
	if err != nil {
		return err
	}
	var allErrors []error
	for _, object := range allObjects {
		err := m.Client.Delete(ctx, object)
		if err != nil {
			allErrors = append(allErrors, err)
		} else {
			logger.Info(
				"deleted resource",
				"name",
				object.GetName(),
				"namespace",
				object.GetNamespace(),
				"kind",
				object.GetObjectKind().GroupVersionKind(),
			)
		}
	}
	if len(allErrors) > 0 {
		return errors.Join(allErrors...)
	}
	return nil
}