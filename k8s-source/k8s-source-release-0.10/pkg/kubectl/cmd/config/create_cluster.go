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
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

type createClusterOptions struct {
	pathOptions           *pathOptions
	name                  string
	server                util.StringFlag
	apiVersion            util.StringFlag
	insecureSkipTLSVerify util.BoolFlag
	certificateAuthority  util.StringFlag
}

func NewCmdConfigSetCluster(out io.Writer, pathOptions *pathOptions) *cobra.Command {
	options := &createClusterOptions{pathOptions: pathOptions}

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("set-cluster name [--%v=server] [--%v=path/to/certficate/authority] [--%v=apiversion] [--%v=true]", clientcmd.FlagAPIServer, clientcmd.FlagCAFile, clientcmd.FlagAPIVersion, clientcmd.FlagInsecure),
		Short: "Sets a cluster entry in .kubeconfig",
		Long: `Sets a cluster entry in .kubeconfig
	Specifying a name that already exists will merge new fields on top of existing values for those fields.
	e.g. 
		kubectl config set-cluster e2e --certificate-authority=~/.kube/e2e/.kubernetes.ca.cert
		only sets the certificate-authority field on the e2e cluster entry without touching other values.
		`,
		Run: func(cmd *cobra.Command, args []string) {
			if !options.complete(cmd) {
				return
			}

			err := options.run()
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		},
	}

	options.insecureSkipTLSVerify.Default(false)

	cmd.Flags().Var(&options.server, clientcmd.FlagAPIServer, clientcmd.FlagAPIServer+" for the cluster entry in .kubeconfig")
	cmd.Flags().Var(&options.apiVersion, clientcmd.FlagAPIVersion, clientcmd.FlagAPIVersion+" for the cluster entry in .kubeconfig")
	cmd.Flags().Var(&options.insecureSkipTLSVerify, clientcmd.FlagInsecure, clientcmd.FlagInsecure+" for the cluster entry in .kubeconfig")
	cmd.Flags().Var(&options.certificateAuthority, clientcmd.FlagCAFile, clientcmd.FlagCAFile+" for the cluster entry in .kubeconfig")

	return cmd
}

func (o createClusterOptions) run() error {
	err := o.validate()
	if err != nil {
		return err
	}

	config, filename, err := o.pathOptions.getStartingConfig()
	if err != nil {
		return err
	}

	if config.Clusters == nil {
		config.Clusters = make(map[string]clientcmdapi.Cluster)
	}

	cluster := o.modifyCluster(config.Clusters[o.name])
	config.Clusters[o.name] = cluster

	err = clientcmd.WriteToFile(*config, filename)
	if err != nil {
		return err
	}

	return nil
}

// cluster builds a Cluster object from the options
func (o *createClusterOptions) modifyCluster(existingCluster clientcmdapi.Cluster) clientcmdapi.Cluster {
	modifiedCluster := existingCluster

	if o.server.Provided() {
		modifiedCluster.Server = o.server.Value()
	}
	if o.apiVersion.Provided() {
		modifiedCluster.APIVersion = o.apiVersion.Value()
	}
	if o.insecureSkipTLSVerify.Provided() {
		modifiedCluster.InsecureSkipTLSVerify = o.insecureSkipTLSVerify.Value()
	}
	if o.certificateAuthority.Provided() {
		modifiedCluster.CertificateAuthority = o.certificateAuthority.Value()
	}

	return modifiedCluster
}

func (o *createClusterOptions) complete(cmd *cobra.Command) bool {
	args := cmd.Flags().Args()
	if len(args) != 1 {
		cmd.Help()
		return false
	}

	o.name = args[0]
	return true
}

func (o createClusterOptions) validate() error {
	if len(o.name) == 0 {
		return errors.New("You must specify a non-empty cluster name")
	}

	return nil
}
