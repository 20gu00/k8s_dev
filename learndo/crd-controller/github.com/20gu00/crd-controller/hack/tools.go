//go:build tools
// +build tools

// 建立 tools.go 来依赖 code-generator
// 因为在没有代码使用 code-generator 时，go module 默认不会为我们依赖此包.

package tools

import _ "k8s.io/code-generator" //会使用的init  自定义标记,tools.go会被自定义工具忽略
