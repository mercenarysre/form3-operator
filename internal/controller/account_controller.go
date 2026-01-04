/*
Copyright 2026.

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

	"github.com/form3tech-oss/go-form3/v7/pkg/form3"
	"github.com/form3tech-oss/go-form3/v7/pkg/generated/models"
	"github.com/go-openapi/strfmt"
	accountv1 "github.com/mercenarysre/forma-operator/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const accountFinalizer = "account.form3.tech/finalizer"

// AccountReconciler reconciles a Account object
type AccountReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Form3Client *form3.F3
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// +kubebuilder:rbac:groups=account.form3.tech,resources=accounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=account.form3.tech,resources=accounts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=account.form3.tech,resources=accounts/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Account object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *AccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	account := &accountv1.Account{}
	if err := r.Get(ctx, req.NamespacedName, account); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Delete Logic
	if !account.DeletionTimestamp.IsZero() {
		if containsString(account.Finalizers, accountFinalizer) {

			if account.Status.ID != "" {
				_, err := r.Form3Client.Accounts.
					DeleteAccount().
					WithContext(ctx).
					WithID(strfmt.UUID(account.Status.ID)).
					Do()

				if err != nil {
					logger.Error(err, "failed to delete Form3 account")
					account.Status.State = "Failed"
					account.Status.Message = err.Error()
					_ = r.Status().Update(ctx, account)
					return ctrl.Result{}, err
				}
			}

			account.Finalizers = removeString(account.Finalizers, accountFinalizer)
			if err := r.Update(ctx, account); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !containsString(account.Finalizers, accountFinalizer) {
		account.Finalizers = append(account.Finalizers, accountFinalizer)
		if err := r.Update(ctx, account); err != nil {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// Create Logic
	if account.Status.State == "" {
		account.Status.State = "Pending"
		account.Status.Message = "Creating Form3 account"
		if err := r.Status().Update(ctx, account); err != nil {
			return ctrl.Result{}, err
		}

		orgID := strfmt.UUID(account.Spec.OrganisationID)
		country := account.Spec.Country

		form3Account := &models.Account{
			Type:           "accounts",
			OrganisationID: &orgID,
			Attributes: &models.AccountAttributes{
				Country:    &country,
				BankID:     &account.Spec.BankID,
				BankIDCode: &account.Spec.BankIDCode,
				Bic:        &account.Spec.BIC,
			},
		}

		resp, err := r.Form3Client.Accounts.
			CreateAccount().
			WithContext(ctx).
			WithData(form3Account).
			Execute()
		if err != nil {
			logger.Error(err, "failed to create Form3 account")
			account.Status.State = "Failed"
			account.Status.Message = err.Error()
			_ = r.Status().Update(ctx, account)
			return ctrl.Result{}, err
		}

		account.Status.ID = resp.Data.ID.String()
		account.Status.IBAN = resp.Data.IBAN
		account.Status.AccountNumber = resp.Data.AccountNumber
		account.Status.BaseCurrency = resp.Data.BaseCurrency
		account.Status.State = "Ready"
		account.Status.Message = "Account successfully created"

		if err := r.Status().Update(ctx, account); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&accountv1.Account{}).
		Named("account").
		Complete(r)
}
