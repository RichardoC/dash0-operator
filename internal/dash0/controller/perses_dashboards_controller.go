// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	otelmetric "go.opentelemetry.io/otel/metric"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/dash0hq/dash0-operator/internal/dash0/util"
)

type PersesDashboardCrdReconciler struct {
	AuthToken                 string
	mgr                       ctrl.Manager
	skipNameValidation        bool
	persesDashboardReconciler *PersesDashboardReconciler
}

//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

type validationResult struct {
	namespace string
	name      string
	url       string
	origin    string
	authToken string
}

var (
	// persesDashboardCrdReconcileRequestMetric otelmetric.Int64Counter
	persesDashboardReconcileRequestMetric otelmetric.Int64Counter

	retrySettings = wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   1.5,
		Steps:    3,
	}
)

func (r *PersesDashboardCrdReconciler) SetupWithManager(
	ctx context.Context,
	mgr ctrl.Manager,
	startupK8sClient client.Client,
	logger *logr.Logger,
) error {
	kubeSystemNamespace := &corev1.Namespace{}
	if err := startupK8sClient.Get(ctx, client.ObjectKey{Name: "kube-system"}, kubeSystemNamespace); err != nil {
		msg := "unable to get the kube-system namespace uid"
		logger.Error(err, msg)
		return fmt.Errorf("%s: %w", msg, err)
	}

	r.mgr = mgr
	r.persesDashboardReconciler = &PersesDashboardReconciler{
		pseudoClusterUid: kubeSystemNamespace.UID,
		httpClient:       &http.Client{},
		authToken:        r.AuthToken,
	}

	if err := startupK8sClient.Get(ctx, client.ObjectKey{
		Name: "persesdashboards.perses.dev",
	}, &apiextensionsv1.CustomResourceDefinition{}); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("The persesdashboards.perses.dev custom resource definition does not exist in this " +
				"cluster, the operator will not watch for Perses dashboard resources.")
		} else {
			logger.Error(err, "unable to call get the persesdashboards.perses.dev custom resource definition")
			return err
		}
	} else {
		logger.Info("The persesdashboards.perses.dev custom resource definition is present in this " +
			"cluster, the operator will watch for Perses dashboard resources.")
		if err = r.startWatchingPersesDashboardResources(ctx, logger); err != nil {
			return err
		}
	}

	// For now, we are not watching for the PersesDashboard CRD. Watching for a foreign CRD and reacting appropriately
	// to its creation/deletion is work in progress in the prometheus scraping branch. Once that is finished, we can
	// employ the same approach here.
	return nil
}

//+kubebuilder:rbac:groups=perses.dev,resources=persesdashboards,verbs=get;list;watch

func (r *PersesDashboardCrdReconciler) InitializeSelfMonitoringMetrics(
	meter otelmetric.Meter,
	metricNamePrefix string,
	logger *logr.Logger,
) {
	// Note: The persesDashboardCrdReconcileRequestMetric is unused until we actually implement watching the
	// PersesDashboard _CRD_, see comment above in SetupWithManager.

	// reconcileRequestMetricName := fmt.Sprintf("%s%s", metricNamePrefix, "persesdashboardcrd.reconcile_requests")
	// var err error
	// if persesDashboardCrdReconcileRequestMetric, err = meter.Int64Counter(
	// 	reconcileRequestMetricName,
	// 	otelmetric.WithUnit("1"),
	// 	otelmetric.WithDescription("Counter for persesdashboard CRD reconcile requests"),
	// ); err != nil {
	// 	logger.Error(err, "Cannot initialize the metric %s.")
	// }

	r.persesDashboardReconciler.InitializeSelfMonitoringMetrics(
		meter,
		metricNamePrefix,
		logger,
	)
}

func (r *PersesDashboardCrdReconciler) startWatchingPersesDashboardResources(
	_ context.Context,
	logger *logr.Logger,
) error {
	logger.Info("Setting up a watch for Perses dashboard custom resources.")

	unstructuredGvkForPersesDashboards := &unstructured.Unstructured{}
	unstructuredGvkForPersesDashboards.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "PersesDashboard",
		Group:   "perses.dev",
		Version: "v1alpha1",
	})

	controllerBuilder := ctrl.NewControllerManagedBy(r.mgr).
		Named("dash0_perses_dashboard_controller").
		Watches(
			unstructuredGvkForPersesDashboards,
			// Deliberately not using a convenience mechanism like &handler.EnqueueRequestForObject{} (which would
			// feed all events into the Reconcile method) here, since using the lower-level TypedEventHandler interface
			// directly allows us to distinguish between create and delete events more easily.
			r.persesDashboardReconciler,
		)
	if r.skipNameValidation {
		controllerBuilder = controllerBuilder.WithOptions(controller.TypedOptions[reconcile.Request]{
			SkipNameValidation: ptr.To(true),
		})
	}
	if err := controllerBuilder.Complete(r.persesDashboardReconciler); err != nil {
		logger.Error(err, "unable to create a new controller for watching Perses Dashboards")
		return err
	}
	r.persesDashboardReconciler.isWatching = true

	return nil
}

func (r *PersesDashboardCrdReconciler) SetApiEndpointAndDataset(apiConfig *ApiConfig) {
	r.persesDashboardReconciler.apiConfig.Store(apiConfig)
}

func (r *PersesDashboardCrdReconciler) RemoveApiEndpointAndDataset() {
	r.persesDashboardReconciler.apiConfig.Store(nil)
}

type ApiConfig struct {
	Endpoint string
	Dataset  string
}

type PersesDashboardReconciler struct {
	isWatching       bool
	pseudoClusterUid types.UID
	httpClient       *http.Client
	apiConfig        atomic.Pointer[ApiConfig]
	authToken        string
}

func (r *PersesDashboardReconciler) InitializeSelfMonitoringMetrics(
	meter otelmetric.Meter,
	metricNamePrefix string,
	logger *logr.Logger,
) {
	reconcileRequestMetricName := fmt.Sprintf("%s%s", metricNamePrefix, "persesdashboard.reconcile_requests")
	var err error
	if persesDashboardReconcileRequestMetric, err = meter.Int64Counter(
		reconcileRequestMetricName,
		otelmetric.WithUnit("1"),
		otelmetric.WithDescription("Counter for perses dashboard reconcile requests"),
	); err != nil {
		logger.Error(err, "Cannot initialize the metric %s.")
	}
}

func (r *PersesDashboardReconciler) Create(
	ctx context.Context,
	e event.TypedCreateEvent[client.Object],
	_ workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	if persesDashboardReconcileRequestMetric != nil {
		persesDashboardReconcileRequestMetric.Add(ctx, 1)
	}

	logger := log.FromContext(ctx)
	logger.Info(
		"Detected a new Perses dashboard resource",
		"namespace",
		e.Object.GetNamespace(),
		"name",
		e.Object.GetName(),
	)
	if err := r.UpsertDashboard(e.Object.(*unstructured.Unstructured), &logger); err != nil {
		logger.Error(err, "unable to upsert the dashboard")
	}
}

func (r *PersesDashboardReconciler) Update(
	ctx context.Context,
	e event.TypedUpdateEvent[client.Object],
	_ workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	if persesDashboardReconcileRequestMetric != nil {
		persesDashboardReconcileRequestMetric.Add(ctx, 1)
	}

	logger := log.FromContext(ctx)
	logger.Info(
		"Detected a change for a Perses dashboard resource",
		"namespace",
		e.ObjectNew.GetNamespace(),
		"name",
		e.ObjectNew.GetName(),
	)

	_ = util.RetryWithCustomBackoff(
		"upsert dashboard",
		func() error {
			return r.UpsertDashboard(e.ObjectNew.(*unstructured.Unstructured), &logger)
		},
		retrySettings,
		true,
		&logger,
	)
}

func (r *PersesDashboardReconciler) Delete(
	ctx context.Context,
	e event.TypedDeleteEvent[client.Object],
	_ workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	if persesDashboardReconcileRequestMetric != nil {
		persesDashboardReconcileRequestMetric.Add(ctx, 1)
	}

	logger := log.FromContext(ctx)
	logger.Info(
		"Detected the deletion of a Perses dashboard resource",
		"namespace",
		e.Object.GetNamespace(),
		"name",
		e.Object.GetName(),
	)

	_ = util.RetryWithCustomBackoff(
		"delete dashboard",
		func() error {
			return r.DeleteDashboard(e.Object.(*unstructured.Unstructured), &logger)
		},
		retrySettings,
		true,
		&logger,
	)
}

func (r *PersesDashboardReconciler) Generic(
	_ context.Context,
	_ event.TypedGenericEvent[client.Object],
	_ workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// ignoring generic events
}

func (r *PersesDashboardReconciler) Reconcile(
	context.Context,
	reconcile.Request,
) (reconcile.Result, error) {
	// Reconcile should not be called on the PersesDashboardReconciler, as we are using the TypedEventHandler interface
	// directly when setting up the watch. We still need to implement the method, as the controller builder's Complete
	// method requires implementing the Reconciler interface.
	return reconcile.Result{}, nil
}

func (r *PersesDashboardReconciler) UpsertDashboard(
	persesDashboard *unstructured.Unstructured,
	logger *logr.Logger,
) error {
	apiConfig := r.apiConfig.Load()
	valResult, executeRequest := r.validateConfigAndRenderUrl(
		persesDashboard,
		apiConfig,
		logger,
	)
	if !executeRequest {
		return nil
	}

	specRaw := persesDashboard.Object["spec"]
	if specRaw == nil {
		logger.Info("Perses dashboard has no spec, the dashboard will not be updated in Dash0.")
		return nil
	}
	spec, ok := specRaw.(map[string]interface{})
	if !ok {
		logger.Info("Perses dashboard spec is not a map, the dashboard will not be updated in Dash0.")
		return nil
	}
	displayRaw := spec["display"]
	if displayRaw == nil {
		spec["display"] = map[string]interface{}{}
		displayRaw = spec["display"]
	}
	display, ok := displayRaw.(map[string]interface{})
	if !ok {
		logger.Info("Perses dashboard spec.display is not a map, the dashboard will not be updated in Dash0.")
		return nil
	}

	displayName, ok := display["name"]
	if !ok || displayName == "" {
		// Let the dashboard name default to the perses dashboard resource's namespace + name, if unset.
		display["name"] = fmt.Sprintf("%s/%s", valResult.namespace, valResult.name)
	}

	// Remove all unnecessary metadata (labels & annotations), we basically only need the dashboard spec.
	serializedDashboard, _ := json.Marshal(
		map[string]interface{}{
			"kind": "PersesDashboard",
			"spec": spec,
		})
	requestPayload := bytes.NewBuffer(serializedDashboard)

	req, err := http.NewRequest(
		http.MethodPut,
		valResult.url,
		requestPayload,
	)
	if err != nil {
		logger.Error(err, "unable to create a new HTTP request to upsert the dashboard")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", valResult.authToken))
	logger.Info(fmt.Sprintf("Updating/creating dashboard %s in Dash0", valResult.origin))
	res, err := r.httpClient.Do(req)
	if err != nil {
		logger.Error(err, fmt.Sprintf("unable to execute the HTTP request to update the dashboard %s", valResult.origin))
		return err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return r.handleNon2xxStatusCode(res, valResult.origin, logger)
	}

	// http status code was 2xx, discard the response body and close it
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	return nil
}

func (r *PersesDashboardReconciler) DeleteDashboard(
	persesDashboard *unstructured.Unstructured,
	logger *logr.Logger,
) error {
	apiConfig := r.apiConfig.Load()
	valResult, executeRequest := r.validateConfigAndRenderUrl(
		persesDashboard,
		apiConfig,
		logger,
	)
	if !executeRequest {
		return nil
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		valResult.url,
		nil,
	)
	if err != nil {
		logger.Error(err, "unable to create a new HTTP request to delete the dashboard")
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", valResult.authToken))
	logger.Info(fmt.Sprintf("Deleting dashboard %s in Dash0", valResult.origin))
	res, err := r.httpClient.Do(req)
	if err != nil {
		logger.Error(err, fmt.Sprintf("unable to execute the HTTP request to delete the dashboard %s", valResult.origin))
		return err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return r.handleNon2xxStatusCode(res, valResult.origin, logger)
	}

	// http status code was 2xx, discard the response body and close it
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	return nil
}

func (r *PersesDashboardReconciler) validateConfigAndRenderUrl(
	persesDashboard *unstructured.Unstructured,
	apiConfig *ApiConfig,
	logger *logr.Logger,
) (*validationResult, bool) {
	if apiConfig == nil || apiConfig.Endpoint == "" {
		logger.Info("No Dash0 API endpoint has been provided via the operator configuration resource, the dashboard " +
			"will not be updated in Dash0.")
		return nil, false
	}
	if r.authToken == "" {
		logger.Info("No auth token is set on the controller deployment, the dashboard will not be updated " +
			"in Dash0.")
		return nil, false
	}

	dataset := apiConfig.Dataset
	if dataset == "" {
		dataset = "default"
	}

	namespace, name, ok := readNamespaceAndName(persesDashboard, logger)
	if !ok {
		return nil, false
	}

	dashboardUrl, dashboardOrigin := r.renderDashboardUrl(
		apiConfig.Endpoint,
		namespace,
		name,
		dataset,
	)
	return &validationResult{
		namespace: namespace,
		name:      name,
		url:       dashboardUrl,
		origin:    dashboardOrigin,
		authToken: r.authToken,
	}, true
}

func (r *PersesDashboardReconciler) renderDashboardUrl(
	dash0ApiEndpoint string,
	namespace string,
	name string,
	dataset string,
) (string, string) {

	dashboardOrigin := fmt.Sprintf(
		// we deliberately use _ as the separator, since that is an illegal character in Kubernetes names. This avoids
		// any potential naming collisions (e.g. namespace="abc" & name="def-ghi" vs. namespace="abc-def" & name="ghi").
		"dash0-operator_%s_%s_%s_%s",
		r.pseudoClusterUid,
		dataset,
		namespace,
		name,
	)
	if !strings.HasSuffix(dash0ApiEndpoint, "/") {
		dash0ApiEndpoint += "/"
	}
	return fmt.Sprintf(
		"%sapi/dashboards/%s?dataset=%s",
		dash0ApiEndpoint,
		dashboardOrigin,
		dataset,
	), dashboardOrigin
}

func (r *PersesDashboardReconciler) handleNon2xxStatusCode(
	res *http.Response,
	dashboardOrigin string,
	logger *logr.Logger,
) error {
	defer func() {
		_ = res.Body.Close()
	}()
	responseBody, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		readBodyErr := fmt.Errorf("unable to read the API response payload after receiving status code %d when "+
			"trying to udpate/create/delete the dashboard %s", res.StatusCode, dashboardOrigin)
		logger.Error(readBodyErr, "unable to read the API response payload")
		return readBodyErr
	}

	statusCodeErr := fmt.Errorf(
		"unexpected status code %d when updating/creating/deleting the dashboard %s, response body is %s",
		res.StatusCode,
		dashboardOrigin,
		string(responseBody),
	)
	logger.Error(statusCodeErr, "unexpected status code")
	return statusCodeErr
}

func readNamespaceAndName(persesDashboard *unstructured.Unstructured, logger *logr.Logger) (string, string, bool) {
	metadataRaw := persesDashboard.Object["metadata"]
	if metadataRaw == nil {
		logger.Info("Perses dashboard payload has no metadata section, the dashboard will not be updated in Dash0.")
		return "", "", false
	}
	metadata, ok := metadataRaw.(map[string]interface{})
	if !ok {
		logger.Info("Perses dashboard payload metadata section is not a map, the dashboard will not be updated in " +
			"Dash0.")
		return "", "", false
	}
	namespace, ok := readStringAttribute(metadata, "namespace", logger)
	if !ok {
		return "", "", false
	}
	name, ok := readStringAttribute(metadata, "name", logger)
	if !ok {
		return "", "", false
	}
	return namespace, name, true
}

func readStringAttribute(metadata map[string]interface{}, attributeName string, logger *logr.Logger) (string, bool) {
	valueRaw := metadata[attributeName]
	if valueRaw == nil {
		logger.Info(fmt.Sprintf("Perses dashboard has no attribute metadata.%s, the dashboard will not be updated in "+
			"Dash0.", attributeName))
		return "", false
	}
	value, ok := valueRaw.(string)
	if !ok {
		logger.Info(fmt.Sprintf("Perses dashboard metadata.%s is not a string, the dashboard will not be updated "+
			"in Dash0.", attributeName))
		return "", false
	}
	if value == "" {
		logger.Info(fmt.Sprintf("Perses dashboard has no attribute metadata.%s, the dashboard will not be updated in "+
			"Dash0.", attributeName))
		return "", false
	}
	return value, true
}
