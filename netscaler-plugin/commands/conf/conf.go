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

package conf

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/constant"
	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/kubectl"
	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/request"
	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/util"
)

// ConfCmdFlag struct for cobra command arguments for conf sub command
type ConfCmdFlag struct {
	pod        *string
	deployment *string
	selector   *string
}

// initConfCmdFlag initializes struct ConfCmdFlag based on json based constants
func initConfCmdFlag(flag *ConfCmdFlag, cmd *cobra.Command) {
	flag.pod = util.AddFlagStringP(cmd, []byte(constant.PodFlag))
	flag.deployment = util.AddFlagStringP(cmd, []byte(constant.DeployFlag))
	flag.selector = util.AddFlagStringP(cmd, []byte(constant.SelectorFlag))
}

// CreateCommand creates the cobra commands for conf subcommand
func CreateCommand(flags *genericclioptions.ConfigFlags) *cobra.Command {
	confCmdFlag := ConfCmdFlag{}
	cmd := &cobra.Command{
		Use:   "conf",
		Short: "Display NetScaler configuration (show run output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			util.PrintError(conf(flags, confCmdFlag))
			return nil
		},
	}
	initConfCmdFlag(&confCmdFlag, cmd)
	return cmd
}

// conf receives user inputs and filters kubernetes object and runs plugin file with conf sub in cic container
func conf(flags *genericclioptions.ConfigFlags, confCmdFlag ConfCmdFlag) error {
	/*********************************************************
	 * kClient cannot be initialized in main to be in sync   *
	 * with Cobra behaviour for defered flags init post RunE *
	 *********************************************************/
	kClient, err := request.NewK8sClient(flags)
	if err != nil {
		fmt.Println("unable to init K8s Client: " + err.Error())
		os.Exit(1)

	}
	pod, cicContainer, _, err := kClient.ChoosePod(flags, *confCmdFlag.pod, *confCmdFlag.deployment, *confCmdFlag.selector)
	if err != nil {
		return err
	}
	validVer, mismatch, err := kubectl.ValidVersion(flags, pod, cicContainer)
	util.CmdErrorHandling(err)
	if !validVer {
		fmt.Print(mismatch)
		return nil
	}
	var flagCommand []string
	if cicContainer == "" {
		flagCommand = []string{"--", constant.PyCmd, constant.PluginFile, "-c", constant.ConfSub}
	} else {
		flagCommand = []string{"-c", cicContainer, "--", constant.PyCmd, constant.PluginFile, "-c", constant.ConfSub}
	}
	op, err := kubectl.PodExecString(flags, &pod, flagCommand)
	if op != "" {
		fmt.Print("\n" + op)
	}
	util.CmdErrorHandling(err)
	return nil
}
