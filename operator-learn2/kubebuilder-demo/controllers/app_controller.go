/*
Copyright 2022.

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
	"github.com/kubebuilder-demo/controllers/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ingressv1beta1 "github.com/kubebuilder-demo/api/v1beta1"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ingress.zqa.demo,resources=apps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.zqa.demo,resources=apps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ingress.zqa.demo,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the App object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	app := &ingressv1beta1.App{}
	//从缓存中获取app
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	//根据app的配置进行处理
	//1. Deployment的处理

	//根据deploy的模板获取deploy
	deployment := utils.NewDeployment(app)
	//设置deploy和app的级联关系
	if err := controllerutil.SetOwnerReference(app, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	//查找同名deploy
	d := &v1.Deployment{}
	//从cache中获取deploy
	if err := r.Get(ctx, req.NamespacedName, d); err != nil {
		if errors.IsNotFound(err) {
			if r.Create(ctx, deployment); err != nil {
				logrus.Error(err, "create deploy failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if err := r.Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
	}

	//处理service
	service := utils.NewService(app)
	//service与app建立级联关系
	if err := controllerutil.SetOwnerReference(app, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	s := &corev1.Service{}
	//查找指定service
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: req.Namespace}, s); err != nil {
		//如果没查到server，并且service的开关被打开了则创建service
		if errors.IsNotFound(err) && app.Spec.EnableService {
			if r.Create(ctx, service); err != nil {
				logrus.Error(err, "create service failed")
				return ctrl.Result{}, err
			}
		}
		//service存在
	} else {
		//更新service
		if app.Spec.EnableService {
			if err := r.Update(ctx, service); err != nil {
				return ctrl.Result{}, err
			}
			//开关未被打开 ，删除service
		} else {
			if err := r.Delete(ctx, service); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	//3. Ingress的处理,service配置可能为空
	//判断service的开关是否存在，如果为false，则没必要创建ingress
	if !app.Spec.EnableService {
		return ctrl.Result{}, nil

	}
	ingress := utils.NewIngress(app)
	//设置app与ingress的级联关系
	if err := controllerutil.SetControllerReference(app, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	i := &netv1.Ingress{}
	//查找ingress
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, i); err != nil {
		//如果找不到ingress并且ingress开关为打开，则创建ingress
		if errors.IsNotFound(err) && app.Spec.EnableIngress {
			if err := r.Create(ctx, ingress); err != nil {
				logrus.Error(err, "create ingress failed")
				return ctrl.Result{}, err
			}
		}

	} else {
		if app.Spec.EnableIngress {
			logrus.Info("skip update")
		} else {
			if err := r.Delete(ctx, i); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//注册资源的监听，监控事件的变化
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1beta1.App{}).
		Owns(&v1.Deployment{}).
		Owns(&netv1.Ingress{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
