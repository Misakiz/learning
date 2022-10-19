package utils

import (
	"bytes"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"text/template"
)

func parseTemplate(templateName string, deployment *appv1.Deployment) []byte {
	tmpl, err := template.ParseFiles("pkg/template/" + templateName + ".yml")
	if err != nil {
		panic(err)
	}
	b := new(bytes.Buffer)
	err = tmpl.Execute(b, deployment)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

// NewService 构建service yaml
func NewService(deployment *appv1.Deployment) *corev1.Service {
	s := &corev1.Service{}

	err := yaml.Unmarshal(parseTemplate("service", deployment), s)
	if err != nil {
		panic(err)
	}
	return s
}
