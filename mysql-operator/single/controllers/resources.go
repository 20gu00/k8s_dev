package controllers

import (
	v1 "github.com/20gu00/mysql-single-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	MysqlSingleCommonLabelKey = "cjqapp"
)

func MutateDeployment(mysqlSingle *v1.MysqlSingle, deploy *appsv1.Deployment) {
	deploy.Labels = map[string]string{
		//deploy的label
		MysqlSingleCommonLabelKey: "mysqlsingle",
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: mysqlSingle.Spec.Replicas,
		Selector: &metav1.LabelSelector,
	}
}
