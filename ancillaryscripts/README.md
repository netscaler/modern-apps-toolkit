# List potential stale servers from the NetScaler

This script lists potential stale server (IP-based) entries present on the NetScaler appliance in a file named `rmserver_<IP>.txt`. 
It is executable both from a remote machine and directly within the NetScaler appliance's shell.

## Pre-requisites

- Python 3.6+
- `requests` module [pip install requests]
- Network connectivity between the remote machine and the NetScaler

## Usage

Run the script with the following command:

```
python list_potential_stale_servers.py <NSIP> <username> <password>
where 
NSIP = NetScaler Management IP address
username = NetScaler username
password = NetScaler password
```

For example,

A) To run the script from a remote machine on NS having management IP '192.168.1.10' and credentials as dummyuser and dummypwd.

`python list_potential_stale_servers.py 192.168.1.10 dummyuser dummypwd`

B) To run the script from the NetScaler shell, localhost IP can be provided as NSIP.

`python list_potential_stale_servers.py 127.0.0.1 dummyuser dummypwd`

This script will generate a rmserver_<IP>.txt file that contains commands to remove "potential" stale server entries. 

**Note:** Take a thorough look at the generated commands before executing them. Ensure that the mentioned server IP addresses are indeed stale. If any server entry seems inappropriate, edit the file (`rmserver_<IP>.txt`) and remove the associated 'rm server' command.

Once the file is reviewed, the below command can be run from the NS CLI to delete stale server entries. 

```> batch -f <path>/rmserver_<IP>.txt```
