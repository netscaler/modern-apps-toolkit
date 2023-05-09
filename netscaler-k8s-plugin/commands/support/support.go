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

package support

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/constant"
	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/kubectl"
	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/request"
	"github.com/netscaler/modern-apps-toolkit/netscaler-k8s-plugin/util"
)

// Constant map for kubernetes objects to query using kubectl get
func getCMD() []string {
	return []string{"pods", "deployment", "svc", "ing", "events", "nodes"}
}

// SupportCmdFlag struct for cobra command arguments for support sub command
type SupportCmdFlag struct {
	pod              *string
	deployment       *string
	selector         *string
	dir              *string
	appns            *string
	skipNsBundleFlag *bool
	unMask           *bool
}

// initSupportCmdFlag initializes struct SupportCmdFlag based on json based constants
func initSupportCmdFlag(flag *SupportCmdFlag, cmd *cobra.Command) {
	flag.pod = util.AddFlagStringP(cmd, []byte(constant.PodFlag))
	flag.deployment = util.AddFlagStringP(cmd, []byte(constant.DeployFlag))
	flag.selector = util.AddFlagStringP(cmd, []byte(constant.SelectorFlag))
	flag.dir = util.AddFlagStringP(cmd, []byte(constant.DirFlag))
	flag.appns = util.AddFlagStringP(cmd, []byte(constant.AppNSFlag))
	flag.skipNsBundleFlag = util.AddFlagBoolP(cmd, []byte(constant.SkipNSBundleFlag))
	flag.unMask = util.AddFlagBoolP(cmd, []byte(constant.UnmaskFlag))
}

// Constant map for kubernetes objects to query using kubectl describe
func descCMD() []string {
	return []string{"svc", "ing", "events", "nodes"}
}

// kubeExtractInfo runs kubectl get and desc for predefined objects
func kubeExtractInfo(flags *genericclioptions.ConfigFlags, ns string, directory string, unMask bool) error {
	for _, cmd := range getCMD() {
		err := kubectl.RunCmdSaveFile(flags, ns, []string{"get", cmd, "-o", "yaml"}, false, directory, unMask)
		if err != nil {
			return err
		}
	}
	for _, cmd := range descCMD() {
		err := kubectl.RunCmdSaveFile(flags, ns, []string{"describe", cmd}, false, directory, unMask)
		if err != nil {
			return err
		}
	}
	return nil
}

// kubeGetLogs runs kubectl get log for running/dead cic container
func kubeGetLogs(flags *genericclioptions.ConfigFlags, podName string, ns string, cicContainer string, directory string, unMask bool) error {
	var logCic, shutLogCic []string
	podName = "pod/" + podName
	logCic = []string{"logs", podName, cicContainer}
	shutLogCic = []string{"logs", "-p", podName, cicContainer}
	err := kubectl.RunCmdSaveFile(flags, ns, []string{"get", podName, "-o", "yaml"}, true, directory, unMask)
	if err != nil {
		fmt.Println("Error while getting Deployments for running CIC")
		return err
	}
	err = kubectl.RunCmdSaveFile(flags, ns, logCic, true, directory, unMask)
	if err != nil {
		fmt.Println("Error while getting logs for running CIC")
	}
	err = kubectl.RunCmdSaveFile(flags, ns, shutLogCic, true, directory, unMask)
	if err != nil {
		fmt.Println("Error while getting logs for shutdown CIC")
	}
	return nil
}

// support receives details of user dir and kube object filters and kubeExtractInfo and kubeGetLogs functions
func troubleShoot(flags *genericclioptions.ConfigFlags, podName string, appns string, ns string, cicContainer string, directory string, unMask bool) error {
	for _, appn := range strings.Fields(appns) {
		err := kubeExtractInfo(flags, appn, directory, unMask)
		if err != nil {
			return err
		}
	}
	err := kubeGetLogs(flags, podName, ns, cicContainer, directory, unMask)
	if err != nil {
		return err
	}
	return nil
}

// CreateCommand creates the cobra commands for support subcommand
func CreateCommand(flags *genericclioptions.ConfigFlags) *cobra.Command {
	supportCmdFlag := SupportCmdFlag{}
	cmd := &cobra.Command{
		Use:   "support",
		Short: "Get NetScaler (show techsupport) and Ingress Controller support bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			util.PrintError(support(flags, supportCmdFlag))
			return nil
		},
	}
	initSupportCmdFlag(&supportCmdFlag, cmd)
	return cmd
}

// support receives user inputs and filters kubernetes object and runs plugin file with support sub in cic container
func support(flags *genericclioptions.ConfigFlags, supportCmdFlag SupportCmdFlag) error {
	/*********************************************************
	 * kClient cannot be initialized in main to be in sync   *
	 * with Cobra behaviour for defered flags init post RunE *
	 *********************************************************/
	kClient, err := request.NewK8sClient(flags)
	if err != nil {
		fmt.Println("unable to init K8s Client: " + err.Error())
		os.Exit(1)

	}
	var flagCommand []string
	pod, cicContainer, cpxContainer, err := kClient.ChoosePod(flags, *supportCmdFlag.pod, *supportCmdFlag.deployment, *supportCmdFlag.selector)
	if err != nil {
		return err
	}
	dir := *supportCmdFlag.dir
	if len(dir) == 0 {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	validVer, mismatch, err := kubectl.ValidVersion(flags, pod, cicContainer)
	util.CmdErrorHandling(err)
	if !validVer {
		fmt.Print(mismatch)
		return nil
	}
	t := time.Now().UTC()
	dir = dir + "/" + constant.DirPrefix + t.Format(constant.DateFormat)

	if !(*supportCmdFlag.skipNsBundleFlag) {
		if len(cicContainer) > 0 {
			flagCommand = []string{"-c", cicContainer, "--", constant.PyCmd, constant.PluginFile, "-c", constant.SupportSub}
		} else {
			flagCommand = []string{"-c", cpxContainer, "--", constant.PyCmd, constant.PluginFile, "-c", constant.SupportSub}
		}
		fmt.Print("Extracting show tech support information, this may take minutes")
		op, err := kubectl.PodExecString(flags, &pod, flagCommand)
		if op != "" {
			fmt.Print("\n" + op)
		}
		util.CmdErrorHandling(err)
		if len(cicContainer) > 0 {
			flagCommand = []string{"--", constant.ReadLinkCmd, "-f", constant.StsSymLink}
			dirNameOP, err := kubectl.PodExecString(flags, &pod, flagCommand)
			util.CmdErrorHandling(err)
			dirNameSlice := strings.Split(dirNameOP, "\n/")
			dirName := strings.TrimSpace(dirNameSlice[len(dirNameSlice)-1])
			flagCommand = []string{pod.Name + ":" + dirName, dir + "/" + constant.StsOp}
			_, err = kubectl.PodCPString(flags, &pod, flagCommand)
			if err != nil {
				return err
			}
		} else {
			fmt.Print("\n" + constant.NonCICSTSComment + constant.StsSymLink)
		}
	} else {
		fmt.Println(constant.NoSTSComment)
	}
	fmt.Println("\nExtracting Kubernetes information")

	err = troubleShoot(flags, pod.Name, *supportCmdFlag.appns, pod.Namespace, cicContainer, dir+"/"+"kube_info", *supportCmdFlag.unMask)
	if err != nil {
		return err
	}
	fmt.Println("The support files are present in " + dir)
	return nil
}
