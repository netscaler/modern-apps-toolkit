# Copyright 2022 Citrix Systems, Inc
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash

disclaimer(){
    echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
    echo "********************************************************************"
    echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
    echo "This script is collecting information related to the Citrix Ingress "
    echo "Controller and applications deployed in the cluster. This script by "
    echo "default masks the IP addresses in the collected output files. The "
    echo "output files will be available in tar format. If there is any information "
    echo "that user deems sensitive and not to be shared, kindly scan through the "
    echo "output_<timestamp> directory under the user provided output directory "
    echo "path and recreate the tar to share."
    echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
    echo "********************************************************************"
    echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
}

get_cic_dep_info() {
    namespace=$1
    namespace_dir=$2
    dep_name=$3
    get_cic_dep_cmd="kubectl get deployment $dep_name -o yaml -n $namespace"
    get_cic_dep_file="cic_deployment/cic_deployments.yaml"
    mkdir -p "$namespace_dir/cic_deployment"
    echo "Collecting $get_cic_dep_cmd output"
    $get_cic_dep_cmd > $namespace_dir/$get_cic_dep_file
    sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$get_cic_dep_file
}

get_pod_info() {
    namespace=$1
    namespace_dir=$2
    get_pods_cmd="kubectl get pods -n $namespace"
    get_pod_file="pod/pods.txt"
    mkdir -p "$namespace_dir/pod"
    echo "Collecting $get_pods_cmd output"
    $get_pods_cmd > $namespace_dir/$get_pod_file
    sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$get_pod_file
}

get_deployment_info() {
    namespace=$1
    namespace_dir=$2
    get_deps_cmd="kubectl get deployment -n $namespace -o yaml"
    get_dep_file="deployment/deployments.yaml"
    mkdir -p "$namespace_dir/deployment"
    echo "Collecting $get_deps_cmd output"
    $get_deps_cmd > $namespace_dir/$get_dep_file
    sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$get_dep_file
}

get_endpoint_info() {
    namespace=$1
    namespace_dir=$2
    desc_ep_cmd="kubectl describe endpoints -n $namespace"
    desc_ep_file="endpoint/endpoints.txt"
    mkdir -p "$namespace_dir/endpoint"
    echo "Collecting $desc_ep_cmd output"
    $desc_ep_cmd > $namespace_dir/$desc_ep_file
    sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$desc_ep_file
}

get_svc_info() {
    namespace=$1
    namespace_dir=$2
    get_svc_cmd="kubectl get svc -o yaml -n $namespace"
    desc_svc_cmd="kubectl describe svc -n $namespace"
    get_svc_file="svc/services.yaml"
    desc_svc_file="svc/desc_services.txt"
    mkdir -p "$namespace_dir/svc"
    echo "Collecting $get_svc_cmd output"
    $get_svc_cmd > $namespace_dir/$get_svc_file
    sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_svc_file

    echo "Collecting $desc_svc_cmd output"
    $desc_svc_cmd > $namespace_dir/$desc_svc_file
    sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$desc_svc_file
} 

get_ingress_info() {
    namespace=$1
    namespace_dir=$2
    get_ingress_cmd="kubectl get ing -n $namespace"
    desc_ingress_cmd="kubectl describe ing -n $namespace"
    get_ingress_file=ingress/ingresses.txt
    desc_ingress_file=ingress/desc_ingresses.txt
    mkdir -p "$namespace_dir/ingress"
    echo "Collecting $get_ingress_cmd output"
    $get_ingress_cmd > $namespace_dir/$get_ingress_file
    sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_ingress_file
    echo "Collecting $desc_ingress_cmd output"
    $desc_ingress_cmd > $namespace_dir/$desc_ingress_file
    sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$desc_ingress_file
}

get_crd_info() {
    supported_crds=`kubectl get crd | grep citrix.com | awk '{print $1}' | tr '\n' ' '`
    mkdir -p "$out_dir/crd_definitions"
    for supported_crd in $supported_crds
    do
        get_crd_cmd="kubectl get crd $supported_crd -o yaml"
        get_crd_file="$out_dir/crd_definitions/$supported_crd.yaml"
        echo "Collecting $get_crd_cmd output"
        $get_crd_cmd > $get_crd_file
    done
}

get_crd_instances() {
    namespace=$1
    namespace_dir=$2
    deployed_crds=`kubectl get crd | grep citrix.com | awk '{print $1}' | tr '\n' ' '`
    mkdir -p "$namespace_dir/crd_instances"
    for crd_instance in $deployed_crds
    do
        get_crd_cmd="kubectl get $crd_instance -n $namespace -o yaml"
        get_crd_file=crd_instances/$crd_instance.yaml
        echo "Collecting $get_crd_cmd output"
        $get_crd_cmd > $namespace_dir/$get_crd_file
        sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_crd_file
    done
}

get_event_info() {
    namespace=$1
    namespace_dir=$2
    get_event_cmd="kubectl get events -n $namespace"
    get_event_file=events/events.txt
    mkdir -p "$namespace_dir/events"
    echo "Collecting $get_event_cmd output"
    $get_event_cmd > $namespace_dir/$get_event_file
    sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_event_file
}

get_node_info() {
    get_nodes_cmd="kubectl describe nodes"
    get_nodes_file=$out_dir/desc_nodes.txt
    touch $get_nodes_file
    echo "Collecting $get_nodes_cmd output"
    $get_nodes_cmd > $get_nodes_file
    sed -i -e $REPLACE_IP_PATTERN $get_nodes_file

    get_nodes_yaml_cmd="kubectl get nodes -o yaml"
    get_nodes_yaml_file=$out_dir/nodes.yaml
    touch $get_nodes_yaml_file
    echo "Collecting $get_nodes_yaml_cmd output"
    $get_nodes_yaml_cmd > $get_nodes_yaml_file
    sed -i -e $REPLACE_IP_PATTERN $get_nodes_yaml_file
} 

get_cic_logs() {
    namespace=$1
    namespace_dir=$2
    dep_name=$3
    container_name=$4
    get_cic_log_cmd="kubectl logs deployment/$dep_name -c $container_name -n $namespace"
    cic_log_file=cic_logs/cic_logs.txt
    mkdir -p "$namespace_dir/cic_logs"
    echo "Collecting CIC logs"
    $get_cic_log_cmd > $namespace_dir/$cic_log_file
    sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$cic_log_file
}

get_restarted_cic_logs(){
    namespace=$1
    namespace_dir=$2
    dep_name=$3
    container_name=$4
    get_cic_log_cmd="kubectl logs -p deployment/$dep_name -c $container_name -n $namespace"
    cic_log_file=cic_logs/restarted_pod_logs.txt
    mkdir -p "$namespace_dir/cic_logs"
    echo "Collecting Restarted CIC logs"
    $get_cic_log_cmd > $namespace_dir/$cic_log_file
    sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$cic_log_file
}

create_tar() {
    cd $user_dir
    zip_file="logs.$timestamp.tar.gz"
    tar -czvf $zip_file -P $output_dir_name
    echo "You can view the logs at " $out_dir
    echo "The zip file of all the logs are available in " $user_dir/$zip_file
} 

disclaimer
echo "****************************************"
echo "Started collecting kubectl outputs!!!"
echo "Collecting kubectl outputs of Pods, Services, Ingress, CRD, Events and CIC logs"
echo "****************************************"
echo "Which CNI has been installed in the cluster?"
read cluster_cni
echo "Enter space separated namespace/deployment/container_name of CIC deployment:(eg: ns1/dep1/cic ns2/dep2/citrix-ingress-controller) "
read cic_dep
echo "Enter space separated namespace(s) of application deployment where ingress, services, pods and crds are deployed:(eg: namespace1 namespace2 namespace3) "
read app_namespace
echo "Enter the absolute path of the directory to collect outputs: "
read user_dir

timestamp=$(date "+%F-%H-%M-%S")
echo "Current time:" + $timestamp
output_dir_name="outputs_$timestamp"
out_dir="$user_dir/$output_dir_name"
mkdir -p $out_dir

REPLACE_IP_PATTERN="s/[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}/x.x.x.x/g"
for ns in $app_namespace
do
    namespace_dir=$out_dir/$ns
    echo "Creating directory for namespace" $ns
    mkdir -p $namespace_dir
    #Getting pod details
    get_pod_info $ns $namespace_dir
    #Getting deployment details
    get_deployment_info $ns $namespace_dir
    # Getting service details
    get_svc_info $ns $namespace_dir
    # Getting ingress details
    get_ingress_info $ns $namespace_dir
    #Getting Endpoints details
    get_endpoint_info $ns $namespace_dir
    # Getting CRD instance details
    get_crd_instances $ns $namespace_dir
    # Getting event details
    get_event_info $ns $namespace_dir
done

for dep in $cic_dep
do
    dep_info=(${dep//\// })
    ns=${dep_info[0]}         
    dep_name=${dep_info[1]}         
    container_name=${dep_info[2]}
    # Getting pod details
    get_cic_dep_info $ns $out_dir/$ns $dep_name
    # Getting CIC logs
    get_cic_logs $ns $out_dir/$ns $dep_name $container_name
    get_restarted_cic_logs $ns $out_dir/$ns $dep_name $container_name
done
# Getting CRD details
get_crd_info
# Getting nodes details
get_node_info

# Get cni details
echo $cluster_cni > $out_dir/cni.txt

# Create tar file
create_tar
