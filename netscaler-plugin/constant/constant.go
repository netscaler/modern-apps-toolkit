package constant

// Constants file storing constants shared across modules
const (
	VersionFile        = "/usr/src/triton/VERSION"
	VersionSupportFrom = "1.31.4"
	PyCmd              = "python3"
	SupportSub         = "support"
	ConfSub            = "conf"
	StatusSub          = "status"
	ReadLinkCmd        = "readlink"
	PluginFile         = "/usr/src/triton/plugin/plugin.py"
	StsSymLink         = "/var/tmp/support/support.tgz"
	StsOp              = "showtechsupport.tgz"
	DateFormat         = "20060102150405"
	DirPrefix          = "nssupport_"
	CicLogsDir         = "cic_logs"
	CicDeployDir       = "cic_deployment"
	CicDeployFile      = "cic_deployment.txt"
	CicLogsFile        = "cic_logs.txt"
	CicLogsRestart     = "restarted_pod_logs.txt"
	RegexpMaskIP       = `((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}`
	NoSTSComment       = "Skipping show tech support collection. For show tech support on NetScaler rerun with by removing --skip-nsbundle option"
	NonCICSTSComment   = "This CIC is connected to VPX/MPX NetScaler appliance. Please securely download support artifact from NetScaler location: "
	UnMaskedIP         = `X.X.X.X`

	/*************************WARNING*************************
	 * Please make sure to maintain the same name for json   *
	 * elements in this file and util/util.go struct CmdFlag *
	 *********************************************************/
	PodFlag    = `{"CmdLName": "pod", "CmdSName": "","DefValueStr": "", "CmdDesc": "Name of the ingress controller pod"}`
	DeployFlag = `{"CmdLName": "deployment", "CmdSName": "","DefValueStr": "", "CmdDesc": "Name of the ingress controller deployment"}`
	//Label rename to
	SelectorFlag     = `{"CmdLName": "label", "CmdSName": "l","DefValueStr": "", "CmdDesc": "Label of the ingress controller deployment"}`
	OutputFlag       = `{"CmdLName": "output", "CmdSName": "","DefValueStr": "", "CmdDesc": "Output format. Supported formats are Tabular (default) and Json"}`
	IngressFlag      = `{"CmdLName": "ingress", "CmdSName": "i","DefValueStr": "", "CmdDesc": "Specify the option to retrieve config status of a particular Kubernetes Ingress Resource"}`
	PrefixFlag       = `{"CmdLName": "prefix", "CmdSName": "p","DefValueStr": "", "CmdDesc": "Specify the name of the Prefix provided while deploying the Ingress Controller"}`
	VerboseFlag      = `{"CmdLName": "verbose", "CmdSName": "v","DefValueB": false, "CmdDesc": "If this option is set, additional information such as Netscaler config type, service port are displayed."}`
	SkipNSBundleFlag = `{"CmdLName": "skip-nsbundle", "CmdSName": "","DefValueB": false, "CmdDesc": "This option enables to extract techsupport from NetScaler. By default this is set to false"}`
	DirFlag          = `{"CmdLName": "dir", "CmdSName": "d","DefValueStr": "", "CmdDesc": "Specify the absolute path of the directory to store support files. If not provided current directory will be used."}`
	AppNSFlag        = `{"CmdLName": "appns", "CmdSName": "","DefValueStr": "default", "CmdDesc": "List of space separated namespaces (within quotes) from where Kubernetes resource details such as ingress, services, pods and crds are extracted (eg: \" default namespace1 namespace2\")"}`
	UnmaskFlag       = `{"CmdLName": "unhideIP", "CmdSName": "","DefValueB": false, "CmdDesc": "Set this to unhide IPs while collecting Kubernetes information. By default this is set to false."}`
)
