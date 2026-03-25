# NetScaler Debugging Tools for Kubernetes
This repository contains a collection of tools and scripts to help debug NetScaler deployments in the Kubernetes environment. These tools are designed to help diagnose and troubleshoot common issues with NetScaler deployments in Kubernetes, such as load balancing and SSL termination.
## Getting Started
To get started with these tools, you'll need to have access to a Kubernetes cluster with NetScaler deployed. You'll also need to have the necessary permissions to deploy and run pods in the cluster.
### Installation
To install the debugging tools, simply clone the repository to your local machine:
```
git clone https://github.com/netscaler/modern-apps-toolkit.git
```
## Tools Included
- Kubectl **netscaler** plugin:  
NetScaler provides a kubectl plug-in **netscaler** to inspect ingress controller deployments and aids in troubleshooting operations. You can inspect NetScaler config and related Kubernetes components using the subcommands available with this plug-in. For more information on how to use the plugin, see [Kubectl plugin document](netscaler-plugin/README.md)

- NSIC Diagnostics Tool:  
NSIC Diagnostics tool is a simple shell script that collects information related to NetScaler Ingress Controller, NetScaler GSLB Controller, NetScaler IPAM Controller, NetScaler Kubernetes Gateway Controller and applications deployed in the Kubernetes cluster.
For more information on how this tool works, see [Diagnostics Tool](diagnostics_tool/README.md)

- Config Cleanup:  
This script cleans up stale NSIC (NetScaler Ingress Controller) configuration from a NetScaler by name prefix. When NSIC is deleted without first removing the associated Ingress/Gateway resources, it leaves behind stale configuration on the NetScaler. Since other config may exist on the same appliance, a blanket `clear config` is not safe. This script deletes only the resources whose names start with a given prefix (e.g. `k8s-` or `k8s_`), covering all entity types that NSIC creates.
For more information on the tool, see [Config Cleanup](config-cleanup/README.md)
