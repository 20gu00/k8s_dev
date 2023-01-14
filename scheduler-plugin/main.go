package main

import (
	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	_ "k8s.io/component-base/metrics/prometheus/version"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	"math/rand"
	"os"
	plugins "schedulePlugin/plugin"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// withplugin用来注册插件,返回的是一个option,插件的参数,可以自定义
	// 将自定义的插件注入到整体的插件中,合并到默认的插件中
	command := app.NewSchedulerCommand(app.WithPlugin(plugins.Name, plugins.New))

	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	// utilflag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
