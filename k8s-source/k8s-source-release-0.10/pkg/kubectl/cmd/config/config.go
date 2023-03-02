/*
Copyright 2014 Google Inc. All rights reserved.

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

package config

import (
	"io"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
)

type pathOptions struct {
	local         bool
	global        bool
	specifiedFile string
}

func NewCmdConfig(out io.Writer) *cobra.Command {
	pathOptions := &pathOptions{}

	cmd := &cobra.Command{
		Use:   "config <subcommand>",
		Short: "config modifies .kubeconfig files",
		Long:  `config modifies .kubeconfig files using subcommands like "kubectl config set current-context my-context"`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// file paths are common to all sub commands
	cmd.PersistentFlags().BoolVar(&pathOptions.local, "local", true, "use the .kubeconfig in the current directory")
	cmd.PersistentFlags().BoolVar(&pathOptions.global, "global", false, "use the .kubeconfig from "+os.Getenv("HOME"))
	cmd.PersistentFlags().StringVar(&pathOptions.specifiedFile, "kubeconfig", "", "use a particular .kubeconfig file")

	cmd.AddCommand(NewCmdConfigView(out, pathOptions))
	cmd.AddCommand(NewCmdConfigSetCluster(out, pathOptions))
	cmd.AddCommand(NewCmdConfigSetAuthInfo(out, pathOptions))
	cmd.AddCommand(NewCmdConfigSetContext(out, pathOptions))
	cmd.AddCommand(NewCmdConfigSet(out, pathOptions))
	cmd.AddCommand(NewCmdConfigUnset(out, pathOptions))
	cmd.AddCommand(NewCmdConfigUseContext(out, pathOptions))

	return cmd
}

func (o *pathOptions) getStartingConfig() (*clientcmdapi.Config, string, error) {
	filename := ""
	config := clientcmdapi.NewConfig()
	switch {
	case o.global:
		filename = os.Getenv("HOME") + "/.kube/.kubeconfig"
		config = getConfigFromFileOrDie(filename)

	case len(o.specifiedFile) > 0:
		filename = o.specifiedFile
		config = getConfigFromFileOrDie(filename)

	case o.local:
		filename = ".kubeconfig"
		config = getConfigFromFileOrDie(filename)
	}

	return config, filename, nil
}

// getConfigFromFileOrDie tries to read a kubeconfig file and if it can't, it calls exit.  One exception, missing files result in empty configs, not an exit
func getConfigFromFileOrDie(filename string) *clientcmdapi.Config {
	var err error
	config, err := clientcmd.LoadFromFile(filename)
	if err != nil && !os.IsNotExist(err) {
		glog.FatalDepth(1, err)
	}

	if config == nil {
		config = clientcmdapi.NewConfig()
	}

	return config
}

func toBool(propertyValue string) (bool, error) {
	boolValue := false
	if len(propertyValue) != 0 {
		var err error
		boolValue, err = strconv.ParseBool(propertyValue)
		if err != nil {
			return false, err
		}
	}

	return boolValue, nil
}
