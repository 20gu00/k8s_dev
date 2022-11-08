package controllers

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// 定义的执行动作接口
type Action interface {
	Execute(context.Context) error
}

// PatchStatus 用户更新对象 status 状态
type PatchStatus struct {
	client   client.Client
	original runtime.Object
	new      runtime.Object
}

func (o *PatchStatus) Execute(ctx context.Context) error {
	if reflect.DeepEqual(o.original, o.new) {
		return nil
	}
	// 更新状态
	if err := o.client.Status().Patch(ctx, o.new, client.MergeFrom(o.original)); err != nil {
		return fmt.Errorf("while patching status error %q", err)
	}

	return nil
}

// CreateObject 创建一个新的资源对象
type CreateObject struct {
	client client.Client
	obj    runtime.Object
}

func (o *CreateObject) Execute(ctx context.Context) error {
	if err := o.client.Create(ctx, o.obj); err != nil {
		return fmt.Errorf("error %q while creating object ", err)
	}
	return nil
}
