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

package util

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var versionRegex = regexp.MustCompile(`(\d)+\.(\d)+\.(\d)+.*`)

// Struct CmdFlag for templating cobra command types
type CmdFlag struct {
	/*************************WARNING*************************
	 * Please make sure to maintain the same name for json   *
	 * elements in this struct and constant/constant.go file *
	 ********************************************************/
	CmdLName    string `json:"CmdLName,omitempty"`
	CmdSName    string `json:"CmdSName,omitempty"`
	DefValueStr string `json:"DefValueStr,omitempty"`
	DefValueB   bool   `json:"DefValueB,omitempty"`
	CmdDesc     string `json:"CmdDesc,omitempty"`
}

// AddFlag for String. Receives cobra references and Json byte array
// This function returns the command line arguments of string type
func AddFlagStringP(cmd *cobra.Command, flagDetails []byte) *string {
	cmdStr := ""
	var cmdFlag CmdFlag
	json.Unmarshal(flagDetails, &cmdFlag)
	cmd.Flags().StringVarP(&cmdStr, cmdFlag.CmdLName, cmdFlag.CmdSName, cmdFlag.DefValueStr, cmdFlag.CmdDesc)
	return &cmdStr
}

// AddFlag for Boolean. Receives cobra references and Json byte array
// This function returns the command line arguments of bool type
func AddFlagBoolP(cmd *cobra.Command, flagDetails []byte) *bool {
	cmdBool := false
	var cmdFlag CmdFlag
	json.Unmarshal(flagDetails, &cmdFlag)
	cmd.Flags().BoolVarP(&cmdBool, cmdFlag.CmdLName, cmdFlag.CmdSName, cmdFlag.DefValueB, cmdFlag.CmdDesc)
	return &cmdBool
}

// PrintError receives an error value and prints it if it exists
func PrintError(e error) {
	if e != nil {
		fmt.Println(e)
	}
}

// CmdErrorHandling receives an error value and gracefully exits in case of exitError
func CmdErrorHandling(err error) {
	if _, ok := err.(*exec.ExitError); ok {
		fmt.Println("error while executing commands in the container")
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// Indicator function receives a channel and keeps printing dots every second to notify background process
func Indicator(shutdownCh <-chan struct{}) {
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fmt.Print(".")
		case <-shutdownCh:
			return
		}
	}
}

// ParseVersionString returns the major, minor, and patch numbers of a version string
func ParseVersionString(v string) (int, int, int, error) {
	parts := versionRegex.FindStringSubmatch(v)

	if len(parts) != 4 {
		return 0, 0, 0, fmt.Errorf("could not parse %v as a version string (like 0.20.3)", v)
	}

	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	patch, _ := strconv.Atoi(parts[3])

	return major, minor, patch, nil
}

// InVersionRangeInclusive checks that the middle version is between the other two versions
func InVersionRangeInclusive(start, v string) bool {
	return !isVersionLessThan(v, start)
}

func isVersionLessThan(a, b string) bool {
	aMajor, aMinor, aPatch, err := ParseVersionString(a)
	if err != nil {
		panic(err)
	}

	bMajor, bMinor, bPatch, err := ParseVersionString(b)
	if err != nil {
		panic(err)
	}

	if aMajor != bMajor {
		return aMajor < bMajor
	}

	if aMinor != bMinor {
		return aMinor < bMinor
	}

	return aPatch < bPatch
}

// PodInDeployment returns whether a pod is part of a deployment with the given name
// a pod is considered to be in {deployment} if it is owned by a replicaset with a name of format {deployment}-otherchars
func PodInDeployment(pod apiv1.Pod, deployment string) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Controller == nil || !*owner.Controller || owner.Kind != "ReplicaSet" {
			continue
		}

		if strings.Count(owner.Name, "-") != strings.Count(deployment, "-")+1 {
			continue
		}

		if strings.HasPrefix(owner.Name, deployment+"-") {
			return true
		}
	}
	return false
}

// GetNamespace takes a set of kubectl flag values and returns the namespace we should be operating in
func GetNamespace(flags *genericclioptions.ConfigFlags) (string, error) {
	namespace, _, err := flags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return "", err
	}
	if len(namespace) == 0 {
		return "", fmt.Errorf("namespace not found")
	}
	return namespace, nil
}
