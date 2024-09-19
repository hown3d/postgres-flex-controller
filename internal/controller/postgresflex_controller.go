/*
Copyright 2024.

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

package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	stackitcloudv1alpha1 "github.com/stackitcloud/postgres-flex-controller/api/v1alpha1"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
)

// PostgresFlexReconciler reconciles a PostgresFlex object
type PostgresFlexReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ProjectID      string
	PostgresClient *postgresflex.APIClient
}

// +kubebuilder:rbac:groups=stackit.cloud,resources=postgresflexes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=stackit.cloud,resources=postgresflexes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=stackit.cloud,resources=postgresflexes/finalizers,verbs=update
// +kubebuilder:rbac:groups=,resources=secrets,verbs=create;update;patch;delete;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PostgresFlex object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PostgresFlexReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var postgres stackitcloudv1alpha1.PostgresFlex
	if err := r.Client.Get(ctx, req.NamespacedName, &postgres); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// TODO(user): your logic here
	flavorResp, err := r.PostgresClient.ListFlavors(ctx, r.ProjectID).Execute()
	if err != nil {
		return reconcile.Result{}, err
	}

	flavors := ptr.Deref(flavorResp.Flavors, []postgresflex.Flavor{})
	var foundFlavorId *string
	for _, flavor := range flavors {
		if ptr.Deref(flavor.Cpu, 0) == postgres.Spec.Flavor.CPU && ptr.Deref(flavor.Memory, 0) == postgres.Spec.Flavor.Memory {
			foundFlavorId = flavor.Id
		}
	}

	if foundFlavorId == nil {
		return ctrl.Result{}, fmt.Errorf("no flavor found for cpu = %d and memory = %d", postgres.Spec.Flavor.CPU, postgres.Spec.Flavor.Memory)
	}

	if postgres.Status.Id == "" {
		payload := postgresflex.CreateInstancePayload{
			FlavorId: foundFlavorId,
			Name:     ptr.To(req.NamespacedName.String()),
			Acl: &postgresflex.ACL{
				Items: &[]string{
					"0.0.0.0/0",
				},
			},
			BackupSchedule: ptr.To("00 00 * * *"),
			Replicas:       ptr.To(int64(1)),
			Options:        &map[string]string{},
			Storage: &postgresflex.Storage{
				Class: ptr.To("premium-perf2-stackit"),
			},
			Version: &postgres.Spec.Version,
		}
		resp, err := r.PostgresClient.CreateInstance(ctx, r.ProjectID).CreateInstancePayload(payload).Execute()
		if err != nil {
			return reconcile.Result{}, err
		}
		if resp == nil {
			return reconcile.Result{}, errors.New("postgresflex API returned nil when creating a instance")
		}
		postgres.Status.Id = *resp.Id
		if err := r.Client.Update(ctx, &postgres); err != nil {
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}
		return reconcile.Result{Requeue: true}, nil
	}

	resp, err := r.PostgresClient.GetInstanceExecute(ctx, r.ProjectID, postgres.Status.Id)
	if err != nil {
		return reconcile.Result{}, err
	}

	if *resp.Item.Status != "Running" {
		return reconcile.Result{Requeue: true}, nil
	}

	if postgres.Status.UserId == "" {
		resp, err := r.PostgresClient.CreateUserExecute(ctx, r.ProjectID, postgres.Status.Id)
		if err != nil {
			return reconcile.Result{}, err
		}

		connectionSecretName := fmt.Sprintf("%s-credentials", req.Name)
		connectionSecret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      connectionSecretName,
				Namespace: req.Namespace,
			},
			StringData: map[string]string{
				"uri":      ptr.Deref(resp.Item.Uri, ""),
				"host":     ptr.Deref(resp.Item.Host, ""),
				"username": ptr.Deref(resp.Item.Username, ""),
				"password": ptr.Deref(resp.Item.Password, ""),
				"port":     strconv.FormatInt(ptr.Deref(resp.Item.Port, int64(0)), 10),
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &connectionSecret, func() error { return nil })
		if err != nil {
			return reconcile.Result{}, err
		}
		postgres.Status.ConnectionSecretName = connectionSecretName
		postgres.Status.UserId = *resp.Item.Id
		err = r.Client.Update(ctx, &postgres)
		if err != nil {
			return reconcile.Result{Requeue: true}, client.IgnoreNotFound(err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresFlexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackitcloudv1alpha1.PostgresFlex{}).
		Complete(r)
}
