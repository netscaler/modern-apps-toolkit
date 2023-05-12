# NetScaler Kubernetes kubectl plug-in

NetScaler provides a  `kubectl` plug-in for inspecting ingress controller deployments and perform troubleshooting operations.
You can perform troubleshooting operations using the subcommands available with this plug-in.
The plugin is supported from Citrix ingress controller version 1.32.7 onwards.

## Installation and usage

You can install the `kubectl` plug-in by downloading it from the [NetScaler Modern Apps tool kit repository](https://github.com/netscaler/modern-apps-toolkit/releases) using curl as follows.

For Linux:

        curl -LO https://github.com/netscaler/modern-apps-toolkit/releases/download/v1.0.0-netscaler-k8s-plugin/netscaler-k8s-plugin_1.0.0-netscaler-k8s-plugin_linux_amd64.tar.gz
        gunzip netscaler-k8s-plugin_1.0.0-netscaler-k8s-plugin_linux_amd64.tar.gz
        tar -xvf netscaler-k8s-plugin_1.0.0-netscaler-k8s-plugin_linux_amd64.tar
        chmod +x netscaler-k8s
        sudo mv netscaler-k8s /usr/local/bin/kubectl-netscaler_k8s

For Mac:

        curl -s -L https://github.com/netscaler/modern-apps-toolkit/releases/download/v1.0.0-netscaler-k8s-plugin/netscaler-k8s-plugin_1.0.0-netscaler-k8s-plugin_darwin_amd64.tar.gz | tar xvz -
        chmod +x netscaler-k8s
        sudo mv netscaler-k8s /usr/local/bin/kubectl-netscaler_k8s

**Note:** For Mac, you need to enable [Open a developer app](https://support.apple.com/en-in/HT202491)


For Windows:

        curl.exe -LO https://github.com/netscaler/modern-apps-toolkit/releases/download/v1.0.0-netscaler-k8s-plugin/netscaler-k8s-plugin_1.0.0-netscaler-k8s-plugin_windows_amd64.tar.gz | tar xvz
        
Rename the file netscaler-k8s.exe to kubectl-netscaler_k8s.exe

**Note:** For Windows, you must set you `$PATH` variable to where netscaler_k8s.exe file is extracted.


The following subcommands are available with this plug in:

| Subcommand   | Description |
---------------| --------------
|  `help`      |   Provides more information about the various options. You can also run this command after installation to check if the installation is successful and see what are the commands available |
|  `status`     | Displays the status (up, down, or active) of NetScaler entities for provided prefix input (the default value of the prefix is `k8s`)|
|  `conf`   |  Displays NetScaler configuration (show run output) |
|  `support`  | Gets NetScaler (`show techsupport`) and Ingress controller support bundle.  Extracts support related information from Citrix ADC and ingress controller. Support related information is extracted as two tar.gz files. These two tar files are `show tech support` information from Citrix ADC and Kubernetes related information for troubleshooting where the ingress controller is deployed.|

## Examples for usage of subcommands

### Help command

You can use the `help` command as follows to know about the available commands.

        # kubectl netscaler-k8s  --help

For more information about a subcommand use the `help` command as follows:

        # kubectl netscaler-k8s  <command> --help

### Status command

The status subcommand shows the status of various components of NetScaler created and
managed by ingress controller in the Kubernetes environment.

The components can be filtered based on either both application prefix (NS_APPS_NAME_PREFIX environment variable for Citrix ingress controller pods or the entity Prefix value in the Helm chart) and ingress name or one of them. Default search prefix is `k8s`.

| Flag      |Short form | Description |
|-----------|-----------|-------------|
|deployment |           | Name of the ingress controller deployment. |
|--ingress  | -i        | Specify the option to retrieve the config status of a particular Kubernetes Ingress resource.|
|--label    | -l |Label of the ingress controller deployment. |
|--output   |    |  Output format. Supported formats are tabular (default) and JSON. |
| --pod     |    | Name of the ingress controller pod.  |
|--prefix   | -p | Specify the name of the Prefix provided while deploying the Ingress controller.|
|--verbose  | -v | If this option is set, additional information such as NetScaler configuration type or service port are displayed.|

The following example shows the status of NetScaler components created by ingress controller with the label `app=cic-tier2-citrix-cpx-with-ingress-controller` and the prefix `plugin2` in the NetScaler namespace.

```
        # kubectl netscaler-k8s status -l app=cic-tier2-citrix-cpx-with-ingress-controller -n netscaler -p plugin

        Showing NetScaler components for prefix: plugin2
        NAMESPACE  INGRESS         PORT  RESOURCE          NAME                                                    STATUS   
        --         --              --    Listener          plugin-198.168.0.1_80_http                              up       
        default    --              --    Traffic Policy    plugin-apache2_80_csp_mqwmhc66h3bkd5i4hd224lve7hjfzvoi  active   
        default    --              --    Traffic Action    plugin-apache2_80_csp_mqwmhc66h3bkd5i4hd224lve7hjfzvoi  attached 
        default    plugin-apache2  80    Load Balancer     plugin-apache2_80_lbv_mqwmhc66h3bkd5i4hd224lve7hjfzvoi  up       
        default    plugin-apache2  80    Service           plugin-apache2_80_sgp_mqwmhc66h3bkd5i4hd224lve7hjfzvoi  --       
        default    plugin-apache2  80    Service Endpoint  198.168.0.2                                             up       
        netscaler  --              --    Traffic Policy    plugin-apache2_80_csp_lhmi6gp3aytmvmww3zczp2yzlyoacebl  active   
        netscaler  --              --    Traffic Action    plugin-apache2_80_csp_lhmi6gp3aytmvmww3zczp2yzlyoacebl  attached 
        netscaler  plugin-apache2  80    Load Balancer     plugin-apache2_80_lbv_lhmi6gp3aytmvmww3zczp2yzlyoacebl  up       
        netscaler  plugin-apache2  80    Service           plugin-apache2_80_sgp_lhmi6gp3aytmvmww3zczp2yzlyoacebl  --       
        netscaler  plugin-apache2  80    Service Endpoint  198.168.0.3                                             up
```

### Conf command

This subcommand shows the running configuration information on the NetScaler (`show run output`).
The `l` option is used for querying the label of Citrix ingress controller pod.

| Flag        |Short form | Description |
|-----------  |-----------|-------------|
| --deployment|           | Name of the ingress controller deployment. |
| --label     | -l        | Label of the ingress controller deployment. |
| --pod       |           | Name of the ingress controller pod.  |

The following is a sample output for the kubectl netscaler-k8s conf subcommand:

```
        # kubectl netscaler-k8s conf -l app=cic-tier2-citrix-cpx-with-ingress-controller -n netscaler

          set ns config -IPAddress 198.168.0.4 -netmask 255.255.255.255
          set ns weblogparam -bufferSizeMB 3
          enable ns feature LB CS SSL REWRITE RESPONDER AppFlow CH
          enable ns mode L3 USNIP PMTUD
          set system user nsroot -encrypted
          set rsskeytype -rsstype ASYMMETRIC
          set lacp -sysPriority 32768 -mac 8a:e6:40:7c:7f:47
          set ns hostName cic-tier2-citrix-cpx-with-ingress-controller-7bf9c46cb9-xpwvm
          set interface 0/1 -haHeartbeat OFF -throughput 0 -bandwidthHigh 0 -bandwidthNormal 0 -intftype Linux -ifnum 0/1
          set interface 0/2 -speed 1000 -duplex FULL -throughput 0 -bandwidthHigh 0 -bandwidthNormal 0 -intftype Linux -ifnum 0/2
```

## Support command

This support subcommand gets NetScaler (show techsupport) and Ingress Controller
support bundle.

**Warning:**
 For tier 2 NetScaler, technical support bundle files are copied to the location the user
specifies. For security reasons, if the ingress controller is managing a tier 1 NetScaler then the tech support bundle is extracted only and not copied. The user must get the technical support bundle files from the NetScaler manually.

Flags for support subcommand:

| Flag      |Short form | Description |
|-----------|-----------|-------------|
| --deployment|           | Name of the ingress controller deployment. |
|--label    | -l |Label of the ingress controller deployment. |
|--pod      |    | Name of the ingress controller pod.  |
| --appns   |     | List of space separated namespaces (within quotes) from where Kubernetes resource details such as ingress, services, pods, and crds are extracted (For example,  default "namespace1" "namespace2") (default "default")   |
| --dir|  -d| Specify the absolute path of the directory to store support files. If not provided, the current directory is used.|
|--unhideIP| | Set this to unhide IP addresses while collecting Kubernetes information. By default, this flag is set to `false`. |
|--skip-nsbundle| |This option disables extraction of techsupport from NetScaler. By default, this flag is set to `false`.|

The following is a sample output for the `kubectl netscaler-k8s  support` command.

        # kubectl netscaler-k8s support -l app=cic-tier2-citrix-cpx-with-ingress-controller -n plugin
        Extracting show tech support information, this may take
        minutes.............

        Extracting Kubernetes information
        The support files are present in /root/nssupport_20230410032954