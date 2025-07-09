# Copyright 2025 Citrix Systems, Inc
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
    echo "This script gathers diagnostic information about the NetScaler Ingress"
    echo "Controller, NetScaler GSLB Controller, NetScaler IPAM Controller, or "
    echo "NetScaler Kubernetes Gateway Controller, as well as details of applications"
    echo "deployed in the cluster."
    echo "The collected data will be packaged into a tar file. If the output"
    echo "contains sensitive information, please review the 'output_<timestamp>'"
    echo "directory in the specified output path and recreate the tar file before"
    echo "sharing."
    echo -e "\033[0;33mWarning: This script does not mask IP addresses in the collected output files by default.\033[0m"
    echo -e "\033[0;33mTo enable IP address masking, uncomment the line containing 'sed -i -e REPLACE_IP_PATTERN' in the script.\033[0m"
    echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
    echo "********************************************************************"
    echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
}

get_all_controller_deployments() {
    container_name=$1
    while read -r namespace depname; do
        namespace_array+=("$namespace")
        dep_array+=("$depname")
    done < <($cluster_env get deployments --all-namespaces -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{" "}{range .spec.template.spec.containers[*]}{.name}{" "}{end}{"\n"}{end}' | awk -v cname="$container_name" '{for(i=3;i<=NF;i++) if($i ~ cname) print $1, $2}')
}

get_controller_dep_info() {
    namespace=$1
    namespace_dir=$2
    dep_name=$3
    get_controller_dep_cmd="$cluster_env get deployment $dep_name -o yaml -n $namespace"
    get_controller_dep_file="${dep_name}_deployment/${dep_name}_deployment.yaml"
    mkdir -p "$namespace_dir/${dep_name}_deployment"
    echo "Collecting $get_controller_dep_cmd output"
    $get_controller_dep_cmd > $namespace_dir/$get_controller_dep_file
    #sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$get_controller_dep_file
}

get_pod_info() {
    namespace=$1
    namespace_dir=$2
    get_pods_cmd="$cluster_env get pods -n $namespace"
    get_pod_file="pod/pods.txt"
    mkdir -p "$namespace_dir/pod"
    echo "Collecting $get_pods_cmd output"
    $get_pods_cmd > $namespace_dir/$get_pod_file
    #sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$get_pod_file
}

get_deployment_info() {
    namespace=$1
    namespace_dir=$2
    get_deps_cmd="$cluster_env get deployment -n $namespace -o yaml"
    get_dep_file="deployment/deployments.yaml"
    mkdir -p "$namespace_dir/deployment"
    echo "Collecting $get_deps_cmd output"
    $get_deps_cmd > $namespace_dir/$get_dep_file
    #sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$get_dep_file
}

get_endpoint_info() {
    namespace=$1
    namespace_dir=$2
    desc_ep_cmd="$cluster_env describe endpoints -n $namespace"
    desc_ep_file="endpoint/endpoints.txt"
    mkdir -p "$namespace_dir/endpoint"
    echo "Collecting $desc_ep_cmd output"
    $desc_ep_cmd > $namespace_dir/$desc_ep_file
    #sed -i -e $REPLACE_IP_PATTERN  $namespace_dir/$desc_ep_file
}

get_svc_info() {
    namespace=$1
    namespace_dir=$2
    get_svc_cmd="$cluster_env get svc -o yaml -n $namespace"
    desc_svc_cmd="$cluster_env describe svc -n $namespace"
    get_svc_file="svc/services.yaml"
    desc_svc_file="svc/desc_services.txt"
    mkdir -p "$namespace_dir/svc"
    echo "Collecting $get_svc_cmd output"
    $get_svc_cmd > $namespace_dir/$get_svc_file
    #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_svc_file

    echo "Collecting $desc_svc_cmd output"
    $desc_svc_cmd > $namespace_dir/$desc_svc_file
    #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$desc_svc_file
} 

get_ingress_info() {
    namespace=$1
    namespace_dir=$2
    get_ingress_cmd="$cluster_env get ing -n $namespace"
    desc_ingress_cmd="$cluster_env describe ing -n $namespace"
    get_ingress_file=ingress/ingresses.txt
    desc_ingress_file=ingress/desc_ingresses.txt
    mkdir -p "$namespace_dir/ingress"
    echo "Collecting $get_ingress_cmd output"
    $get_ingress_cmd > $namespace_dir/$get_ingress_file
    #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_ingress_file
    echo "Collecting $desc_ingress_cmd output"
    $desc_ingress_cmd > $namespace_dir/$desc_ingress_file
    # sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$desc_ingress_file
    get_ingress_list=$($cluster_env get ing -n $namespace -o jsonpath='{.items[*].metadata.name}')
    for ingress in $get_ingress_list
    do
        get_ingress_yaml="$cluster_env get ing $ingress -n $namespace -o yaml"
        get_ingress_yaml_file=ingress/$ingress.yaml
        echo "Collecting $get_ingress_yaml output"
        $get_ingress_yaml > $namespace_dir/$get_ingress_yaml_file
        # sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_ingress_yaml_file
    done
}

get_configmap_info() {
    namespace=$1
    namespace_dir=$2
    get_configmap_cmd="$cluster_env get configmap -n $namespace -o yaml"
    get_configmap_file="configmap/configmaps.yaml"
    mkdir -p "$namespace_dir/configmap"
    echo "Collecting $get_configmap_cmd output"
    $get_configmap_cmd > $namespace_dir/$get_configmap_file
    #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_configmap_file
}

get_crd_info() {
    supported_crds=`$cluster_env get crd | grep citrix.com | awk '{print $1}' | tr '\n' ' '`
    mkdir -p "$out_dir/crd_definitions"
    for supported_crd in $supported_crds
    do
        get_crd_cmd="$cluster_env get crd $supported_crd -o yaml"
        get_crd_file="$out_dir/crd_definitions/$supported_crd.yaml"
        echo "Collecting $get_crd_cmd output"
        $get_crd_cmd > $get_crd_file
    done
}

get_crd_instances() {
    namespace=$1
    namespace_dir=$2
    deployed_crds=`$cluster_env get crd | grep citrix.com | awk '{print $1}' | tr '\n' ' '`
    mkdir -p "$namespace_dir/crd_instances"
    for crd_instance in $deployed_crds
    do
        get_crd_cmd="$cluster_env get $crd_instance -n $namespace -o yaml"
        get_crd_file=crd_instances/$crd_instance.yaml
        echo "Collecting $get_crd_cmd output"
        $get_crd_cmd > $namespace_dir/$get_crd_file
        #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_crd_file
    done
}

get_gwy_crd_info() {
    supported_crds=`$cluster_env get crd | grep gateway.networking.k8s.io | awk '{print $1}' | tr '\n' ' '`
    mkdir -p "$out_dir/gateway_crd_definitions"
    for supported_crd in $supported_crds
    do
        get_crd_cmd="$cluster_env get crd $supported_crd -o yaml"
        get_crd_file="$out_dir/gateway_crd_definitions/$supported_crd.yaml"
        echo "Collecting $get_crd_cmd output"
        $get_crd_cmd > $get_crd_file
    done
}

get_gwy_crd_instances() {
    namespace=$1
    namespace_dir=$2
    deployed_crds=`$cluster_env get crd | grep gateway.networking.k8s.io | awk '{print $1}' | tr '\n' ' '`
    mkdir -p "$namespace_dir/gateway_crd_instances"
    for crd_instance in $deployed_crds
    do
        get_crd_cmd="$cluster_env get $crd_instance -n $namespace -o yaml"
        get_crd_file=gateway_crd_instances/$crd_instance.yaml
        echo "Collecting $get_crd_cmd output"
        $get_crd_cmd > $namespace_dir/$get_crd_file
        #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_crd_file
    done
}

get_event_info() {
    namespace=$1
    namespace_dir=$2
    get_event_cmd="$cluster_env get events -n $namespace"
    get_event_file=events/events.txt
    mkdir -p "$namespace_dir/events"
    echo "Collecting $get_event_cmd output"
    $get_event_cmd > $namespace_dir/$get_event_file
    #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$get_event_file
}

get_node_info() {
    get_nodes_cmd="$cluster_env describe nodes"
    get_nodes_file=$out_dir/desc_nodes.txt
    touch $get_nodes_file
    echo "Collecting $get_nodes_cmd output"
    $get_nodes_cmd > $get_nodes_file
    #sed -i -e $REPLACE_IP_PATTERN $get_nodes_file

    get_nodes_yaml_cmd="$cluster_env get nodes -o yaml"
    get_nodes_yaml_file=$out_dir/nodes.yaml
    touch $get_nodes_yaml_file
    echo "Collecting $get_nodes_yaml_cmd output"
    $get_nodes_yaml_cmd > $get_nodes_yaml_file
    #sed -i -e $REPLACE_IP_PATTERN $get_nodes_yaml_file
} 

get_logs() {
    namespace=$1
    namespace_dir=$2
    dep_name=$3
    container_name=$4
    get_log_cmd="$cluster_env logs deployment/$dep_name -c $container_name -n $namespace"
    log_file="logs/${dep_name}_${container_name}_logs.txt"
    mkdir -p "$namespace_dir/logs"
    echo "Collecting $container_name logs"
    $get_log_cmd > $namespace_dir/$log_file
    #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$log_file
}

get_restarted_logs(){
    namespace=$1
    namespace_dir=$2
    dep_name=$3
    container_name=$4
    get_log_cmd="$cluster_env logs -p deployment/$dep_name -c $container_name -n $namespace"
    log_file="logs/restarted_pod_${dep_name}_${container_name}_logs.txt"
    mkdir -p "$namespace_dir/logs"
    echo "Collecting $container_name restarted logs"
    $get_log_cmd > $namespace_dir/$log_file
    #sed -i -e $REPLACE_IP_PATTERN $namespace_dir/$log_file
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
echo "Starting diagnostics collection..."
echo "Gathering information for Pods, Services, Ingresses, CRDs, Events, and Logs."
echo "****************************************"
# Check for kubectl or oc and set cluster_env accordingly
if command -v kubectl >/dev/null 2>&1; then
    cluster_env="kubectl"
elif command -v oc >/dev/null 2>&1; then
    cluster_env="oc"
else
    echo -e "\033[0;31mError: Neither kubectl nor oc is installed or available in PATH.\033[0m"
    echo -e "\033[0;31mPlease install kubectl or oc CLI tool and run the script again.\033[0m"
    exit 1
fi

echo "Specify the CNI installed in your cluster (e.g., Flannel, Calico, Cilium, etc.):"
read cluster_cni

# Controller deployment details
controller_choice_array=()
namespace_array=()
dep_array=()

while true; do
    echo "Select the NetScaler Controller to collect logs for:"
    echo "1. NetScaler Ingress Controller"
    echo "2. NetScaler GSLB Controller"
    echo "3. NetScaler IPAM Controller"
    echo "4. NetScaler Kubernetes Gateway Controller"
    echo "5. All NetScaler Controllers deployed in the cluster"
    read -p "Enter the corresponding number of your choice: " nsic_choice

    case $nsic_choice in
        1)
            controller_choice="NetScaler Ingress Controller"
            container_name="cic|nsic"
            get_all_controller_deployments $container_name
            ;;
        2)
            controller_choice="NetScaler GSLB Controller"
            container_name="gslb"
            get_all_controller_deployments $container_name
            ;;
        3)
            controller_choice="NetScaler IPAM Controller"
            container_name="ipam"
            get_all_controller_deployments $container_name
            ;;
        4)
            controller_choice="NetScaler Kubernetes Gateway Controller"
            container_name="nsgc"
            get_all_controller_deployments $container_name
            ;;
        5)
            controller_choice="All"
            controller_choice_array+=("$controller_choice")
            container_name="cic|nsic|ipam|gslb|nsgc|netscaler"
            get_all_controller_deployments $container_name
            break
            ;;
        *)
            echo "Invalid choice. Please try again."
            continue
            ;;
    esac
    controller_choice_array+=("$controller_choice")

    read -p "Do you want to add more deployments from another NetScaler controller? (yes/no): " more_nsic
    case "$more_nsic" in
        [Yy][Ee][Ss]|[Yy])
            continue
            ;;
        [Nn][Oo]|[Nn])
            break
            ;;
        *)
            echo "Invalid input. Please enter 'yes' or 'no'."
            continue
            ;;
    esac
done

echo "Enter the namespace(s) where your applications (ingress, services, pods, and CRDs) are deployed, separated by spaces (e.g., namespace1 namespace2 namespace3):"
echo "Or press Enter to collect outputs from all namespaces."
echo -e "\033[0;33mWarning: Pressing Enter will collect outputs from every namespace in your cluster.\033[0m"
read app_namespace
echo "Enter the absolute path of the directory to collect outputs: "
read user_dir

timestamp=$(date "+%F-%H-%M-%S")
echo "Current time:" + $timestamp
output_dir_name="outputs_$timestamp"
out_dir="$user_dir/$output_dir_name"
mkdir -p $out_dir

# REPLACE_IP_PATTERN="s/[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}/x.x.x.x/g"
if [ -z "$app_namespace" ]; then
    # If app_namespace is empty, get all namespaces
    app_namespace=$($cluster_env get ns -o jsonpath='{.items[*].metadata.name}')
fi

for ns in $app_namespace
do
    namespace_dir=$out_dir/$ns
    echo "Creating directory for namespace" $ns
    mkdir -p $namespace_dir
    # Getting pod details
    get_pod_info $ns $namespace_dir
    # Getting deployment details
    get_deployment_info $ns $namespace_dir
    # Getting service details
    get_svc_info $ns $namespace_dir
    # Getting ingress details
    get_ingress_info $ns $namespace_dir
    # Getting configmap details
    get_configmap_info $ns $namespace_dir
    # Getting Endpoints details
    get_endpoint_info $ns $namespace_dir
    # Getting CRD instance details
    get_crd_instances $ns $namespace_dir
    # Getting event details
    get_event_info $ns $namespace_dir
done

# Collecting Gateway CRD info only if any controller is "NetScaler Kubernetes Gateway Controller"
# Or if "All" controllers are selected
if printf '%s\n' "${controller_choice[@]}" | grep -Eq "Gateway|All"; then
    get_gwy_crd_info
    for ns in $app_namespace; do
        namespace_dir=$out_dir/$ns
        get_gwy_crd_instances $ns $namespace_dir
    done
fi

for i in ${!dep_array[@]}
do
    dep_name=${dep_array[$i]}
    ns=${namespace_array[$i]}
    container_names=$($cluster_env get deployment $dep_name -o jsonpath='{.spec.template.spec.containers[*].name}' -n $ns)
    # Getting deployment details
    get_controller_dep_info $ns $out_dir/$ns $dep_name
    # Getting logs
    for container_name in $container_names; do
        get_logs $ns $out_dir/$ns $dep_name $container_name
        get_restarted_logs $ns $out_dir/$ns $dep_name $container_name
    done
done
# Getting CRD details
get_crd_info
# Getting nodes details
get_node_info

# Get cni details
echo $cluster_cni > $out_dir/cni.txt

# Create tar file
create_tar
