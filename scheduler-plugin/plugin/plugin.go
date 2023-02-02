package plugins

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

const (
	// Name 定义插件名称
	Name              = "sample-plugin"
	preFilterStateKey = "PreFilter" + Name
)

// prefilter filter
// 实现一个prefilter 插件,模仿scheduler自带的
// prefilterplugin的interface
var _ framework.PreFilterPlugin = &Sample{}
var _ framework.FilterPlugin = &Sample{}

// 调度插件的参数
type SampleArgs struct {
	FavoriteColor  string `json:"favorColor,omitempty"`
	FavoriteNumber int    `json:"favorNumber,omitempty"`
	ThanksTo       string `json:"thanksTo,omitempty"`
}

// 获取插件配置的参数
func getSampleArgs(object runtime.Object) (*SampleArgs, error) {
	// 如果参数格式符合runtime.Object也就是typemeta等那些,直接断言即可,将object传递给参数结构体
	sa := &SampleArgs{}
	// decodeinto将object转换成参数结构体
	if err := frameworkruntime.DecodeInto(object, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

type preFilterState struct {
	framework.Resource // requests,limits
}

// 实现这个函数 实现接口
func (s *preFilterState) Clone() framework.StateData {
	return s
}

func getPreFilterState(state *framework.CycleState) (*preFilterState, error) {
	// 读取出来
	data, err := state.Read(preFilterStateKey)
	if err != nil {
		return nil, err
	}
	s, ok := data.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v convert to SamplePlugin preFilterState error", data)
	}
	return s, nil
}

// 要实现这个prefilter插件,sample要实现以下方法
type Sample struct {
	args   *SampleArgs
	handle framework.FrameworkHandle
}

func (s *Sample) Name() string {
	return Name
}

func computePodResourceLimit(pod *v1.Pod) *preFilterState {
	result := &preFilterState{}
	for _, container := range pod.Spec.Containers {
		// 添加进来
		result.Add(container.Resources.Limits)
	}
	return result
}

func (s *Sample) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	// 日志等级
	if klog.V(2).Enabled() {
		klog.InfoS("Start PreFilter Pod", "pod", pod.Name)
	}
	// key value
	state.Write(preFilterStateKey, computePodResourceLimit(pod))
	return nil
}

func (s *Sample) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (s *Sample) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	preState, err := getPreFilterState(state)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}
	if klog.V(2).Enabled() {
		klog.InfoS("Start Filter Pod", "pod", pod.Name, "node", nodeInfo.Node().Name, "preFilterState", preState)
	}
	// logic
	// 调度错误和错误描述
	return framework.NewStatus(framework.Success, "")
}

//type PluginFactory = func(configuration runtime.Object, f v1alpha1.FrameworkHandle) (v1alpha1.Plugin, error)
// 调度框架
// 初始化插件
func New(object runtime.Object, f framework.FrameworkHandle) (framework.Plugin, error) {
	// 获取参数
	args, err := getSampleArgs(object)
	if err != nil {
		return nil, err
	}
	// validate args
	if klog.V(2).Enabled() {
		klog.InfoS("Successfully get plugin config args", "plugin", Name, "args", args)
	}
	return &Sample{
		args:   args,
		handle: f,
	}, nil
}
