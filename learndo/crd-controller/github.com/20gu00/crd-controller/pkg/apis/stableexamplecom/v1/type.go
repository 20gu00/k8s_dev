package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CronTab struct { // 根据 CRD 定义 CronTab 结构体(crontab.yaml)(生成客户端的代码clienset)(实现Object,所有的k8s的内置资源都会实现object)(这里不用使用status,如果开启status,有Status字段就会生成status信息)
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CronTabSpec `json:"spec"`
}

// +k8s:deepcopy-gen=false

type CronTabSpec struct { //不是顶级api资源
	CronSpec string `json:"cronSpec"`
	Image    string `json:"image"`
	Replicas int    `json:"replicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CronTabList struct { // CronTab 资源列表(专门的list)
	metav1.TypeMeta `json:",inline"`

	// 标准的 list metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CronTab `json:"items"`
}
