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
NSIC Diagnostics tool is a simple shell script that collects information related to NetScaler Ingress Controller and applications deployed in the Kubernetes cluster.
For more information on how this tool, see [NSIC Diagnostics Tool](nsic_diagnostics_tool/README.md)
