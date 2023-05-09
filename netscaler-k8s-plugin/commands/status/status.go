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

package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/constant"
	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/kubectl"
	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/request"
	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/util"
)

// StatusCmdFlag struct for cobra command arguments for status sub command
type StatusCmdFlag struct {
	pod        *string
	deployment *string
	selector   *string
	output     *string
	ing        *string
	prefix     *string
	verbosity  *bool
}

// initStatusCmdFlag initializes struct StatusCmdFlag based on json based constants
func initStatusCmdFlag(flag *StatusCmdFlag, cmd *cobra.Command) {
	flag.pod = util.AddFlagStringP(cmd, []byte(constant.PodFlag))
	flag.deployment = util.AddFlagStringP(cmd, []byte(constant.DeployFlag))
	flag.selector = util.AddFlagStringP(cmd, []byte(constant.SelectorFlag))
	flag.output = util.AddFlagStringP(cmd, []byte(constant.OutputFlag))
	flag.ing = util.AddFlagStringP(cmd, []byte(constant.IngressFlag))
	flag.prefix = util.AddFlagStringP(cmd, []byte(constant.PrefixFlag))
	flag.verbosity = util.AddFlagBoolP(cmd, []byte(constant.VerboseFlag))
}

// CreateCommand creates the cobra commands for status subcommand
func CreateCommand(flags *genericclioptions.ConfigFlags) *cobra.Command {
	statusCmdFlag := StatusCmdFlag{}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display the status (up/down/active...) of NetScaler entities for provided prefix input (Default value of the prefix is k8s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			util.PrintError(status(flags, statusCmdFlag))
			return nil
		},
	}
	initStatusCmdFlag(&statusCmdFlag, cmd)
	return cmd
}

// status receives user inputs and filters kubernetes object and runs plugin file with status sub in cic container
func status(flags *genericclioptions.ConfigFlags, statusCmdFlag StatusCmdFlag) error {
	/*********************************************************
	 * kClient cannot be initialized in main to be in sync   *
	 * with Cobra behaviour for defered flags init post RunE *
	 *********************************************************/
	kClient, err := request.NewK8sClient(flags)
	if err != nil {
		fmt.Println("unable to init K8s Client: " + err.Error())
		os.Exit(1)

	}
	pod, cicContainer, _, err := kClient.ChoosePod(flags, *statusCmdFlag.pod, *statusCmdFlag.deployment, *statusCmdFlag.selector)
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
	lenApp := len(*statusCmdFlag.ing)
	lenPfix := len(*statusCmdFlag.prefix)
	lenOp := len(*statusCmdFlag.output)

	if cicContainer == "" {
		flagCommand = []string{"--", constant.PyCmd, constant.PluginFile, "-c", constant.StatusSub}
	} else {
		flagCommand = []string{"-c", cicContainer, "--", constant.PyCmd, constant.PluginFile, "-c", constant.StatusSub}
	}
	if lenApp > 0 {
		flagCommand = append(flagCommand, "-i", *statusCmdFlag.ing)
	}
	if lenPfix > 0 {
		flagCommand = append(flagCommand, "-p", *statusCmdFlag.prefix)
	}
	if lenOp > 0 {
		flagCommand = append(flagCommand, "-o", *statusCmdFlag.output)
	}
	if *statusCmdFlag.verbosity {
		flagCommand = append(flagCommand, "-v")
	}
	cicStatus, err := kubectl.PodExecString(flags, &pod, flagCommand)
	util.CmdErrorHandling(err)
	fmt.Println(strings.TrimRight(strings.Trim(cicStatus, " \n"), " \n\t"))
	return nil
}
