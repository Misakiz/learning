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


### 创建webhook
> 创建webhook之前需要先创建api


1. 生成代码

```shell
kubebuilder create webhook --group ingress --version v1beta1 --kind App --defaulting --programmatic-validation
 ```


创建之后，在main.go中会添加以下代码:

```go
	if err = (&ingressv1beta1.App{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "App")
		os.Exit(1)
	}
```

同时会生成下列文件，主要有：

- api/v1beta1/app_webhook.go webhook对应的handler，我们添加业务逻辑的地方

- api/v1beta1/webhook_suite_test.go 测试

- config/certmanager 自动生成自签名的证书，用于webhook server提供https服务

- config/webhook 用于注册webhook到k8s中

- config/crd/patches 为conversion自动注入caBoundle

- config/default/manager_webhook_patch.yaml 让manager的deployment支持webhook请求
- config/default/webhookcainjection_patch.yaml 为webhook server注入caBoundle

注入caBoundle由cert-manager的[ca-injector](https://cert-manager.io/docs/concepts/ca-injector/#examples) 组件实现

2. 修改配置

为了支持webhook，我们需要修改config/default/kustomization.yaml将相应的配置打开，具体可参考注释。
```yaml
# Adds namespace to all resources.
namespace: kubebuilder-demo-system

# Value of this field is prepended to the
# names of all resources, e.g. a deployment named
# "wordpress" becomes "alices-wordpress".
# Note that it should also match with the prefix (text before '-') of the namespace
# field above.
namePrefix: kubebuilder-demo-

# Labels to add to all resources and selectors.
#commonLabels:
#  someName: someValue

bases:
- ../crd
- ../rbac
- ../manager
# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in
# crd/kustomization.yaml
- ../webhook
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'. 'WEBHOOK' components are required.
- ../certmanager
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
#- ../prometheus

patchesStrategicMerge:
# Protect the /metrics endpoint by putting it behind auth.
# If you want your controller-manager to expose the /metrics
# endpoint w/o any authn/z, please comment the following line.
- manager_auth_proxy_patch.yaml

# Mount the controller config file for loading manager configurations
# through a ComponentConfig type
#- manager_config_patch.yaml

# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in
# crd/kustomization.yaml
- manager_webhook_patch.yaml

# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'.
# Uncomment 'CERTMANAGER' sections in crd/kustomization.yaml to enable the CA injection in the admission webhooks.
# 'CERTMANAGER' needs to be enabled to use ca injection
- webhookcainjection_patch.yaml

# the following config is for teaching kustomize how to do var substitution
vars:
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
  objref:
    kind: Certificate
    group: cert-manager.io
    version: v1
    name: serving-cert # this name should match the one in certificate.yaml
  fieldref:
    fieldpath: metadata.namespace
- name: CERTIFICATE_NAME
  objref:
    kind: Certificate
    group: cert-manager.io
    version: v1
    name: serving-cert # this name should match the one in certificate.yaml
- name: SERVICE_NAMESPACE # namespace of the service
  objref:
    kind: Service
    version: v1
    name: webhook-service
  fieldref:
    fieldpath: metadata.namespace
- name: SERVICE_NAME
  objref:
    kind: Service
    version: v1
    name: webhook-service

```

### webhook业务逻辑

#### 设置enable_ingress的默认值
```go
func (r *App) Default() {
	applog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	r.Spec.EnableIngress = !r.Spec.EnableIngress
}
```

#### 校验enable_service的值

```go
func (r *App) ValidateCreate() error {
	applog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validateApp()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *App) ValidateUpdate(old runtime.Object) error {
	applog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return r.validateApp()
}

func (r *App) validateApp() error {
	if !r.Spec.EnableService && r.Spec.EnableIngress {
		return apierrors.NewInvalid(GroupVersion.WithKind("App").GroupKind(), r.Name,
			field.ErrorList{
				field.Invalid(field.NewPath("enable_service"),
					r.Spec.EnableService,
					"enable_service should be true when enable_ingress is true"),
			},
		)
	}
	return nil
}
```
### 测试

1. 安装cert-manager

```shell
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml
```

2. 部署

```shell
IMG=wangtaotao2015/app-controller make deploy
```

3. 验证

```yaml
apiVersion: ingress.baiding.tech/v1beta1
kind: App
metadata:
  name: app-sample
spec:
  image: nginx:latest
  replicas: 3
  enable_ingress: false #会被修改为true
  enable_service: false #将会失败

```

```yaml
apiVersion: ingress.baiding.tech/v1beta1
kind: App
metadata:
  name: app-sample
spec:
  image: nginx:latest
  replicas: 3
  enable_ingress: false #会被修改为true
  enable_service: true #成功

```

```yaml
apiVersion: ingress.baiding.tech/v1beta1
kind: App
metadata:
  name: app-sample
spec:
  image: nginx:latest
  replicas: 3
  enable_ingress: true #会被修改为false
  enable_service: false #成功

```

### 如何本地测试

1. 添加本地测试相关的代码

- config/dev

- Makefile

```shell
.PHONY: dev
dev: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/dev | kubectl apply -f -
.PHONY: undev
undev: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/dev | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

```


2. 获取证书放到临时文件目录下

```shell
kubectl get secrets webhook-server-cert -n  kubebuilder-demo-system -o jsonpath='{..tls\.crt}' |base64 -d > certs/tls.crt
kubectl get secrets webhook-server-cert -n  kubebuilder-demo-system -o jsonpath='{..tls\.key}' |base64 -d > certs/tls.key
```

3. 修改main.go,让webhook server使用指定证书

```go
	if os.Getenv("ENVIRONMENT") == "DEV" {
		path, err := os.Getwd()
		if err != nil {
			setupLog.Error(err, "unable to get work dir")
			os.Exit(1)
		}
		options.CertDir = path + "/certs"
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
```

4. 部署

```shell
make dev
```

5. 清理环境

```shell
make undev
```



