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

package request

import (
	"context"
	"errors"
	"fmt"

	"github.com/netscaler/modern-apps-toolkit/netscaler-plugin/util"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sclient "k8s.io/client-go/kubernetes"
)

// Struct K8sClient for common clientset and functions
type K8sClient struct {
	K8sClient *k8sclient.Clientset
}

// NewK8sClient Creates a common k8s clientset for different client types
func NewK8sClient(flags *genericclioptions.ConfigFlags) (K8sClient, error) {
	rawConfig, err := flags.ToRESTConfig()
	kClient := K8sClient{}
	if err != nil {
		kClient.K8sClient = &k8sclient.Clientset{}
		return kClient, err
	}
	clientSet, err := k8sclient.NewForConfig(rawConfig)
	if err != nil {
		kClient.K8sClient = &k8sclient.Clientset{}
		return kClient, err
	}
	kClient.K8sClient = clientSet
	return kClient, nil
}

// ChoosePod finds a pod either by deployment or by name
func (kClient *K8sClient) ChoosePod(flags *genericclioptions.ConfigFlags, podName string, deployment string, selector string) (apiv1.Pod, string, string, error) {
	var pod apiv1.Pod
	var err error
	if podName != "" {
		pod, err = kClient.GetNamedPod(flags, podName)
	} else if selector != "" {
		pod, err = kClient.GetLabeledPod(flags, selector)
	} else if deployment != "" {
		pod, err = kClient.GetDeploymentPod(flags, deployment)
	} else {
		err = errors.New("please provide either label (-l, --label), deployment (--deployment) or pod (--pod ) as a selector in the command")
		return pod, "", "", err
	}
	cicContainer, cpxContainer := "", ""
	if len(pod.Spec.Containers) > 0 {
		for container := range pod.Spec.Containers {
			for env := range pod.Spec.Containers[container].Env {
				if pod.Spec.Containers[container].Env[env].Name == "NS_DEPLOYMENT_MODE" && pod.Spec.Containers[container].Env[env].Value == "SIDECAR" {
					cicContainer = pod.Spec.Containers[container].Name
				} else {
					cpxContainer = pod.Spec.Containers[container].Name
				}
			}
		}
	}
	return pod, cicContainer, cpxContainer, err
}

// GetNamedPod finds a pod with the given name
func (kClient *K8sClient) GetNamedPod(flags *genericclioptions.ConfigFlags, name string) (apiv1.Pod, error) {
	allPods, err := kClient.getPods(flags)
	if err != nil {

		return apiv1.Pod{}, err
	}
	// Running should be constant
	for _, pod := range allPods {
		if pod.Name == name && pod.Status.Phase == "Running" {
			return pod, nil
		}
	}
	namespace, err := util.GetNamespace(flags)
	if err != nil {
		return apiv1.Pod{}, err
	}
	return apiv1.Pod{}, fmt.Errorf("pod %v not found in namespace %v or is not in healthy state", name, namespace)
}

// GetDeploymentPod finds a pod from a given deployment
func (kClient *K8sClient) GetDeploymentPod(flags *genericclioptions.ConfigFlags, deployment string) (apiv1.Pod, error) {
	ings, err := kClient.getDeploymentPods(flags, deployment)
	if err != nil {

		return apiv1.Pod{}, err
	}
	namespace, err := util.GetNamespace(flags)
	if err != nil {
		return apiv1.Pod{}, err
	}
	if len(ings) == 0 {
		return apiv1.Pod{}, fmt.Errorf("no pods for deployment %v found in namespace %v", deployment, namespace)
	}
	// Return the first Running pod
	for _, pod := range ings {
		if pod.Status.Phase == "Running" {
			return pod, nil
		}
	}

	return apiv1.Pod{}, fmt.Errorf("no pods for deployment %v found in namespace %v with healthy state", deployment, namespace)
}

// GetLabeledPod finds a pod from a given label
func (kClient *K8sClient) GetLabeledPod(flags *genericclioptions.ConfigFlags, label string) (apiv1.Pod, error) {
	ings, err := kClient.getLabeledPods(flags, label)
	if err != nil {
		return apiv1.Pod{}, err

	}
	namespace, err := util.GetNamespace(flags)
	if err != nil {
		return apiv1.Pod{}, err
	}

	if len(ings) == 0 {
		return apiv1.Pod{}, fmt.Errorf("no pods for label selector %v found in namespace %v", label, namespace)
	}
	for _, pod := range ings {
		if pod.Status.Phase == "Running" {
			return pod, nil
		}
	}
	return apiv1.Pod{}, fmt.Errorf("no pods for label selector %v found in namespace %v with healthy state", label, namespace)
}

// GetDeployments returns an array of Deployments
func (kClient *K8sClient) GetDeployments(flags *genericclioptions.ConfigFlags, namespace string) ([]appsv1.Deployment, error) {

	deployments, err := kClient.K8sClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {

		return make([]appsv1.Deployment, 0), err
	}

	return deployments.Items, nil
}

// GetIngressDefinitions returns an array of Ingress resource definitions
func (kClient *K8sClient) GetIngressDefinitions(flags *genericclioptions.ConfigFlags, namespace string) ([]networking.Ingress, error) {

	pods, err := kClient.K8sClient.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {

		return make([]networking.Ingress, 0), err
	}

	return pods.Items, nil
}

// GetNumEndpoints counts the number of endpointslices adresses for the service with the given name
func (kClient *K8sClient) GetNumEndpoints(flags *genericclioptions.ConfigFlags, namespace string, serviceName string) (*int, error) {
	epss, err := kClient.GetEndpointSlicesByName(flags, namespace, serviceName)
	if err != nil {

		return nil, err
	}

	if len(epss) == 0 {
		return nil, nil
	}

	ret := 0
	for _, eps := range epss {
		for _, ep := range eps.Endpoints {
			ret += len(ep.Addresses)
		}
	}
	return &ret, nil
}

// GetEndpointSlicesByName returns the endpointSlices for the service with the given name
func (kClient *K8sClient) GetEndpointSlicesByName(flags *genericclioptions.ConfigFlags, namespace string, name string) ([]discoveryv1.EndpointSlice, error) {
	allEndpointsSlices, err := kClient.getEndpointSlices(flags, namespace)
	if err != nil {

		return nil, err
	}
	var eps []discoveryv1.EndpointSlice
	for _, slice := range allEndpointsSlices {
		if svcName, ok := slice.ObjectMeta.GetLabels()[discoveryv1.LabelServiceName]; ok {
			if svcName == name {
				eps = append(eps, slice)
			}
		}
	}

	return eps, nil
}

var endpointSlicesCache = make(map[string]*[]discoveryv1.EndpointSlice)

// getEndpointSlices returns the endpointSlices for the service with the given name
func (kClient *K8sClient) getEndpointSlices(flags *genericclioptions.ConfigFlags, namespace string) ([]discoveryv1.EndpointSlice, error) {
	cachedEndpointSlices, ok := endpointSlicesCache[namespace]

	if ok {
		return *cachedEndpointSlices, nil
	}

	if namespace != "" {
		kClient.tryAllNamespacesEndpointSlicesCache(flags)
	}

	cachedEndpointSlices = kClient.tryFilteringEndpointSlicesFromAllNamespacesCache(flags, namespace)

	if cachedEndpointSlices != nil {
		return *cachedEndpointSlices, nil
	}
	endpointSlicesList, err := kClient.K8sClient.DiscoveryV1().EndpointSlices(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	endpointSlices := endpointSlicesList.Items

	endpointSlicesCache[namespace] = &endpointSlices
	return endpointSlices, nil
}

func (kClient *K8sClient) tryAllNamespacesEndpointSlicesCache(flags *genericclioptions.ConfigFlags) {
	_, ok := endpointSlicesCache[""]
	if !ok {
		_, err := kClient.getEndpointSlices(flags, "")
		if err != nil {
			endpointSlicesCache[""] = nil
		}
	}
}

func (kClient *K8sClient) tryFilteringEndpointSlicesFromAllNamespacesCache(flags *genericclioptions.ConfigFlags, namespace string) *[]discoveryv1.EndpointSlice {
	allEndpointSlices := endpointSlicesCache[""]
	if allEndpointSlices != nil {
		endpointSlices := make([]discoveryv1.EndpointSlice, 0)
		for _, slice := range *allEndpointSlices {
			if slice.Namespace == namespace {
				endpointSlices = append(endpointSlices, slice)
			}
		}
		endpointSlicesCache[namespace] = &endpointSlices
		return &endpointSlices
	}
	return nil
}

// GetServiceByName finds and returns the service definition with the given name
func (kClient *K8sClient) GetServiceByName(flags *genericclioptions.ConfigFlags, name string, services *[]apiv1.Service) (apiv1.Service, error) {
	if services == nil {
		servicesArray, err := kClient.getServices(flags)
		if err != nil {
			return apiv1.Service{}, err
		}
		services = &servicesArray
	}

	for _, svc := range *services {
		if svc.Name == name {
			return svc, nil
		}
	}
	namespace, err := util.GetNamespace(flags)
	if err != nil {
		return apiv1.Service{}, err
	}
	return apiv1.Service{}, fmt.Errorf("could not find service %v in namespace %v", name, namespace)
}

// getPods finds and returns the pods with the given name
func (kClient *K8sClient) getPods(flags *genericclioptions.ConfigFlags) ([]apiv1.Pod, error) {
	namespace, err := util.GetNamespace(flags)
	if err != nil {
		return make([]apiv1.Pod, 0), err
	}
	pods, err := kClient.K8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return make([]apiv1.Pod, 0), err
	}

	return pods.Items, nil
}

// getLabeledPods finds and returns the pods with the given labels
func (kClient *K8sClient) getLabeledPods(flags *genericclioptions.ConfigFlags, label string) ([]apiv1.Pod, error) {
	namespace, err := util.GetNamespace(flags)
	if err != nil {
		return make([]apiv1.Pod, 0), err
	}
	pods, err := kClient.K8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: label,
	})

	if err != nil {
		return make([]apiv1.Pod, 0), err
	}

	return pods.Items, nil
}

// getDeploymentPods finds and returns the pods with the given deployment name
func (kClient *K8sClient) getDeploymentPods(flags *genericclioptions.ConfigFlags, deployment string) ([]apiv1.Pod, error) {
	pods, err := kClient.getPods(flags)
	if err != nil {
		return make([]apiv1.Pod, 0), err
	}

	ingressPods := make([]apiv1.Pod, 0)
	for _, pod := range pods {
		if util.PodInDeployment(pod, deployment) {
			ingressPods = append(ingressPods, pod)
		}
	}

	return ingressPods, nil
}

// getServices finds and returns the pods svc the given namespace name
func (kClient *K8sClient) getServices(flags *genericclioptions.ConfigFlags) ([]apiv1.Service, error) {
	namespace, err := util.GetNamespace(flags)
	if err != nil {
		return make([]apiv1.Service, 0), err
	}
	services, err := kClient.K8sClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return make([]apiv1.Service, 0), err
	}

	return services.Items, nil

}
