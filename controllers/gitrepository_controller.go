/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/codeclysm/extract"
	sourcectrlv1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/makkes/fluxmc/api/v1alpha1"
)

// GitRepositoryReconciler reconciles a GitRepository object
type GitRepositoryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func NewGitRepositoryReconciler(c client.Client, s *runtime.Scheme, log logr.Logger) *GitRepositoryReconciler {
	return &GitRepositoryReconciler{
		Client: c,
		Scheme: s,
		Log:    log,
	}
}

type AppSyncErrors struct {
	errs map[string]error
}

func (e AppSyncErrors) Error() string {
	var b strings.Builder
	for k, v := range e.errs {
		b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	return b.String()
}

//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *GitRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.WithValues("app", req.NamespacedName.String()).Info("reconciling")

	var repo sourcectrlv1beta1.GitRepository
	if err := r.Get(ctx, req.NamespacedName, &repo); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("could not get resource: %w", err)
	}

	if repo.Status.URL == "" {
		// not ready, yet
		return ctrl.Result{}, nil
	}

	resp, err := http.Get(repo.Status.URL)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not download repo artifact: %w", err)
	}
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not create temp dir for repo artifact: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extract.Archive(ctx, resp.Body, tmpDir, nil); err != nil {
		return ctrl.Result{}, fmt.Errorf("could not extract repo artifact: %w", err)
	}

	apps, err := filepath.Glob(path.Join(tmpDir, "apps", "*"))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not glob apps: %w", err)
	}

	errs := make(map[string]error)
	for _, app := range apps {
		appName := path.Base(app)
		appObj := v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name: appName,
			},
			Spec: v1alpha1.ApplicationSpec{
				Repo: v1alpha1.CrossNamespaceGitRepositoryReference{
					Namespace: repo.Namespace,
					Name:      repo.Name,
				},
			},
		}
		r.Log.Info("creating or updating", "app", appName)
		if err := r.Client.Create(ctx, &appObj); err != nil {
			if errors.IsAlreadyExists(err) {
				var appObjFromServer v1alpha1.Application
				if errGet := r.Client.Get(ctx, client.ObjectKeyFromObject(&appObj), &appObjFromServer); errGet != nil {
					errs[appName] = errGet
					continue
				}
				appObjFromServer.Spec = appObj.Spec
				if errUp := r.Client.Update(ctx, &appObjFromServer); errUp != nil {
					errs[appName] = errUp
				}
				continue
			}
			errs[appName] = err
		}
	}

	if len(errs) > 0 {
		return ctrl.Result{}, AppSyncErrors{
			errs: errs,
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sourcectrlv1beta1.GitRepository{}).
		Complete(r)
}
