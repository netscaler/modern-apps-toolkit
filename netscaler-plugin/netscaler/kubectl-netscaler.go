/*
Copyright 2019 The Kubernetes Authors.
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

package main

import (
	"fmt"
	"os"

	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/commands/conf"
	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/commands/status"
	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/commands/support"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// main invoked during plugin binary execution with arguments as a cobra cli app
func main() {

	rootCmd := &cobra.Command{
		Use:   "netscaler",
		Short: "A Kubernetes plugin for inspecting Ingress Controller and associated NetScaler deployments",
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	// Respect some basic kubectl flags like --namespace
	flags := genericclioptions.NewConfigFlags(true)
	flags.AddFlags(rootCmd.PersistentFlags())
	// Add custom subcommands supported by plugin
	rootCmd.AddCommand(status.CreateCommand(flags))
	rootCmd.AddCommand(support.CreateCommand(flags))
	rootCmd.AddCommand(conf.CreateCommand(flags))
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
