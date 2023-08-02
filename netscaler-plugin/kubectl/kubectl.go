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

package kubectl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/constant"
	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/util"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func ValidVersion(flags *genericclioptions.ConfigFlags, pod apiv1.Pod, cicContainer string) (bool, string, error) {
	versionSupportFrom := constant.VersionSupportFrom
	var flagCommand []string
	if cicContainer == "" {
		flagCommand = []string{"--", "cat", constant.VersionFile}
	} else {
		flagCommand = []string{"-c", cicContainer, "--", "cat", constant.VersionFile}
	}
	op, err := PodExecString(flags, &pod, flagCommand)
	if err != nil {
		return false, "", err
	}
	if len(op) == 0 {
		mismatchMsg := "CIC Version file is Empty, CIC Version supported from: " + versionSupportFrom
		return false, mismatchMsg, nil
	}
	if util.InVersionRangeInclusive(versionSupportFrom, op) {
		return true, "", nil
	}
	mismatchMsg := "CIC Version: " + op + " not supported for kubectl plugin. This is supported from: " + versionSupportFrom
	return false, mismatchMsg, nil
}

// maskIP takes any IP as string and converts it into a masked IP format
func maskIP(umaskedS string) string {
	var re = regexp.MustCompile(constant.RegexpMaskIP)
	maskedS := re.ReplaceAllString(umaskedS, constant.UnMaskedIP)
	return maskedS
}

// createDirFile takes user defined directory and creates a directory for support subcommand
func createDirFile(directory string, filename string, content string) error {

	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if len(content) > 0 {
		repoFile := directory + "/" + filename
		repoFile = filepath.Clean(repoFile)
		file, err := os.Create(repoFile)
		if err != nil {
			return err
		}
		_, err = file.WriteString(content)
		if err != nil {
			return err
		}
	}
	return nil
}

// RunCmdSaveFile takes a pod and a command, uses kubectl get and set to extract kube object details
func RunCmdSaveFile(flags *genericclioptions.ConfigFlags, ns string, args []string, flag bool, dir string, unMask bool) error {
	var maskedOut string
	kArgs := getKubectlConfigFlags(flags)
	kArgs = append(kArgs, "-n", ns)
	kArgs = append(kArgs, args...)
	out, err := exec.Command("kubectl", kArgs...).Output()
	if err != nil && args[0] != "logs" {
		fmt.Println(err)
		return err
	}
	absPathCicLog := dir + "/" + ns + "/" + constant.CicLogsDir
	absPathCicDep := dir + "/" + ns + "/" + constant.CicDeployDir
	if unMask {
		maskedOut = string(out)
	} else {
		maskedOut = maskIP(string(out))
	}
	if flag {
		if args[0] == "logs" {
			if args[1] == "-p" {
				err = createDirFile(absPathCicLog, constant.CicLogsRestart, maskedOut)
				if err != nil {
					return err
				}
			} else {
				err = createDirFile(absPathCicLog, constant.CicLogsFile, maskedOut)
				if err != nil {
					return err
				}
			}
		} else {
			err = createDirFile(absPathCicDep, constant.CicDeployFile, maskedOut)
			if err != nil {
				return err
			}
		}

	} else {
		err = createDirFile(dir+"/"+ns+"/"+args[1], args[0]+"_"+args[1]+".txt", maskedOut)
		if err != nil {
			return err
		}
	}
	return nil
}

// PodExecString takes a pod and a command, uses kubectl exec to run the command in the pod
// and returns stdout as a string
func PodExecString(flags *genericclioptions.ConfigFlags, pod *apiv1.Pod, args []string) (string, error) {
	args = append([]string{"exec", "-n", pod.Namespace, pod.Name}, args...)
	return ExecToString(flags, args)
}

func PodCPString(flags *genericclioptions.ConfigFlags, pod *apiv1.Pod, args []string) (string, error) {
	args = append([]string{"cp", "-n", pod.Namespace}, args...)
	return ExecToString(flags, args)
}

// ExecToString runs a kubectl subcommand and returns stdout as a string
func ExecToString(flags *genericclioptions.ConfigFlags, args []string) (string, error) {
	kArgs := getKubectlConfigFlags(flags)
	kArgs = append(kArgs, args...)
	buf := bytes.NewBuffer(make([]byte, 0))
	err := execToWriter(append([]string{"kubectl"}, kArgs...), buf)
	if err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

// Replaces the currently running process with the given command
func execCommand(args []string) error {
	path, err := exec.LookPath(args[0])
	if err != nil {
		return err
	}
	args[0] = path

	env := os.Environ()
	return syscall.Exec(path, args, env)
}

// Replaces the currently running process with the given command and puts to stdout
func execToWriter(args []string, writer io.Writer) error {
	cmd := exec.Command(args[0], args[1:]...)
	op, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err != nil {
		return err
	}
	shutdownCh := make(chan struct{})
	go util.Indicator(shutdownCh)
	go io.Copy(writer, op)
	err = cmd.Run()
	close(shutdownCh)
	if err != nil {
		return err
	}
	return nil
}

// getKubectlConfigFlags serializes the parsed flag struct back into a series of command line args
// that can then be passed to kubectl. The mirror image of
// https://github.com/kubernetes/cli-runtime/blob/master/pkg/genericclioptions/config_flags.go#L251
func getKubectlConfigFlags(flags *genericclioptions.ConfigFlags) []string {
	out := []string{}
	o := &out

	appendStringFlag(o, flags.KubeConfig, "kubeconfig")
	appendStringFlag(o, flags.CacheDir, "cache-dir")
	appendStringFlag(o, flags.CertFile, "client-certificate")
	appendStringFlag(o, flags.KeyFile, "client-key")
	appendStringFlag(o, flags.BearerToken, "token")
	appendStringFlag(o, flags.Impersonate, "as")
	appendStringArrayFlag(o, flags.ImpersonateGroup, "as-group")
	appendStringFlag(o, flags.Username, "username")
	appendStringFlag(o, flags.Password, "password")
	appendStringFlag(o, flags.ClusterName, "cluster")
	appendStringFlag(o, flags.AuthInfoName, "user")
	//appendStringFlag(o, flags.Namespace, "namespace")
	appendStringFlag(o, flags.Context, "context")
	appendStringFlag(o, flags.APIServer, "server")
	appendBoolFlag(o, flags.Insecure, "insecure-skip-tls-verify")
	appendStringFlag(o, flags.CAFile, "certificate-authority")
	appendStringFlag(o, flags.Timeout, "request-timeout")
	return out
}

// parses struct flags in genericCliOptions to string
func appendStringFlag(out *[]string, in *string, flag string) {
	if in != nil && *in != "" {
		*out = append(*out, fmt.Sprintf("--%v=%v", flag, *in))
	}
}

// parses struct flags in genericCliOptions to Boolean
func appendBoolFlag(out *[]string, in *bool, flag string) {
	if in != nil {
		*out = append(*out, fmt.Sprintf("--%v=%v", flag, *in))
	}
}

// parses struct flags in genericCliOptions to StringArray
func appendStringArrayFlag(out *[]string, in *[]string, flag string) {
	if in != nil && len(*in) > 0 {
		*out = append(*out, fmt.Sprintf("--%v=%v'", flag, strings.Join(*in, ",")))
	}
}
