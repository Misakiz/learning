# kubebuilder

init
--domain string            domain for groups (default "my.domain")
create a group

kubebuilder init --domain zqa.demo

kubebuilder create api --group ingress --version v1beta1 --kind App


## 创建结构体
type AppSpec struct {
// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
// Important: Run "make" to regenerate code after modifying this file
EnableIngress bool  `json:"enable_ingress,omitempty"`
EnableService bool  `json:"enable_service "`
Replicas      int32 `json:"replicas"`
// Foo is an example field of App. Edit app_types.go to remove/update
Image string `json:"image"`
}


# webhook

kubebuilder create webhook --group ingress --version v1beta1 --kind App --defaulting --programmatic-validation --conversion


### 重新生成crd资源

```shell
make manifests
```

### 实现Reconcile逻辑

1. App的处理

```go
	logger := log.FromContext(ctx)
	app := &ingressv1beta1.App{}
	//从缓存中获取app
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
```

2. Deployment的处理

之前我们创建资源对象时，都是通过构造golang的struct来构造，但是对于复杂的资源对象
这样做费时费力，所以，我们可以先将资源定义为go template，然后替换需要修改的值之后，
反序列号为golang的struct对象，然后再通过client-go帮助我们创建或更新指定的资源。

我们的deployment、service、ingress都放在了controllers/template中，通过
utils来完成上述过程。

```go
	//根据app的配置进行处理
	//1. Deployment的处理
	deployment := utils.NewDeployment(app)
	if err := controllerutil.SetControllerReference(app, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	//查找同名deployment
	d := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, d); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, deployment); err != nil {
				logger.Error(err, "create deploy failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if err := r.Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
	}
```

3. Service的处理
```go
	//2. Service的处理
	service := utils.NewService(app)
	if err := controllerutil.SetControllerReference(app, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	//查找指定service
	s := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, s); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableService {
			if err := r.Create(ctx, service); err != nil {
				logger.Error(err, "create service failed")
				return ctrl.Result{}, err
			}
		}
		//Fix: 这里还需要修复一下
	} else {
		if app.Spec.EnableService {
			//Fix: 当前情况下，不需要更新，结果始终都一样
			if err := r.Update(ctx, service); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err := r.Delete(ctx, s); err != nil {
				return ctrl.Result{}, err
			}

		}
	}
```

4. Ingress的处理
```go
	//3. Ingress的处理,ingress配置可能为空
	//TODO 使用admission校验该值,如果启用了ingress，那么service必须启用
	//TODO 使用admission设置默认值,默认为false
	//Fix: 这里会导致Ingress无法被删除
	if !app.Spec.EnableService {
		return ctrl.Result{}, nil
	}
	ingress := utils.NewIngress(app)
	if err := controllerutil.SetControllerReference(app, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	i := &netv1.Ingress{}
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, i); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableIngress {
			if err := r.Create(ctx, ingress); err != nil {
				logger.Error(err, "create service failed")
				return ctrl.Result{}, err
			}
		}
		//Fix: 这里还是需要重试一下
	} else {
		if app.Spec.EnableIngress {
            //Fix: 当前情况下，不需要更新，结果始终都一样
			if err := r.Update(ctx, ingress); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err := r.Delete(ctx, i); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
```

5. 删除service、ingress、deployment时，自动重建

```go
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1beta1.App{}).
		Owns(&v1.Deployment{}).
		Owns(&netv1.Ingress{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
```


### 测试

#### 安装ingress controller

我们这里使用traefik作为ingress controller。

```shell
helm install traefik traefik/traefik -f traefik_values.yaml

cat <<EOF>> traefik_values.yaml
ingressClass:
  enabled: true
  isDefaultClass: true #指定为默认的ingress
EOF

helm install traefik traefik/traefik -f traefik_values.yaml 
```
#### 安装crd

```shell
make install
```

