# This script will list down POTENTIAL stale server(IP based) entries on the NetScaler in a file rmserver_<IP>.txt.
# Pre-requisites: Python 3.6+ version, requests module
# Usage: python list_potential_stale_servers.py <NSIP> <username> <password>
# Author: NetScaler Ingress Controller Team

import argparse
import requests
import ipaddress

# REST Nitro API endpoints
LOGIN_URL = "/nitro/v1/config/login"
LOGOUT_URL = "/nitro/v1/config/logout"
NSRUNNING_CONFIG_URL = "/nitro/v1/config/nsrunningconfig"
PARSE_SERVER_STRING = "add server " # Note the space after "server"
PARSE_SERVICE_STRING = "add service " # Note the space after "service"
PARSE_SVCGRP_STRING = "bind servicegroup " # Note the space after "servicegroup"

def is_valid_ip(ipstr):
    try:
        ipaddress.ip_address(ipstr)
        return True
    except ValueError:
        return False

def login(ip, user, password):
    login_payload = {
        "login": {
            "username": user,
            "password": password,
            "session_timeout": 10
        }
    }

    try:
        response = requests.post(f'http://{ip}{LOGIN_URL}', json=login_payload)
        response.raise_for_status()  # Raise HTTPError for bad responses
        return response.json()['sessionid']
    except requests.exceptions.RequestException as e:
        print(f"Login error: {e}")
        return None

def logout(ip, session_id):
    try:
        response = requests.post(f'http://{ip}{LOGOUT_URL}', headers={'Cookie': f'NITRO_AUTH_TOKEN={session_id}'})
        response.raise_for_status()
    except requests.exceptions.RequestException as e:
        print(f"Logout error: {e}")

def get_resources(ip, url, headers):
    try:
        response = requests.get(f'http://{ip}{url}', headers=headers, verify=False)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"Failed to get resources. URL: {url}, Error: {e}")
        return None

def get_runningconfig(ip, headers):
    runconfig = get_resources(ip, NSRUNNING_CONFIG_URL, headers)
    if 'nsrunningconfig' in runconfig:
        resp = runconfig['nsrunningconfig']['response']
        #print(f"\tResponse: {resp}")
    return resp

# Parse runconfig for "add server" entries
def get_server_entries(runconfig):
    servers = set()
    for line in runconfig.splitlines():
        if line.lower().startswith(PARSE_SERVER_STRING):
            parts = line.split()
            server = parts[2]
            if len(parts) > 3 and parts[2] == parts[3] and is_valid_ip(server):
                servers.add(server)
    return servers

# Parse runconfig for "add service" entries
def get_service_entries(runconfig):
    services = set()
    for line in runconfig.splitlines():
        if line.lower().startswith(PARSE_SERVICE_STRING):
            #print(f"\t{line}")
            parts = line.split()
            if len(parts) > 3 and is_valid_ip(parts[3]):
                    services.add(parts[3])
    return services

# Parse runconfig for "bind servicegroup" entries
def get_servicegroupmember_entries(runconfig):
    svcgrpmembers = set()
    for line in runconfig.splitlines():
        if line.lower().startswith(PARSE_SVCGRP_STRING):
            #print(f"\t{line}")
            parts = line.split()
            if len(parts) > 3 and is_valid_ip(parts[3]):
                svcgrpmembers.add(parts[3])
    return svcgrpmembers

# This will generate the file rmserver.txt. 
# This file can be used to remove the stale servers by running NS CLI command: batch -f rmserver.txt
def generate_rm_commands(pot_stale_servers, ip):
    # Write output to rmserver.txt
    # filename to be based on input ip address
    fname = 'rmserver_' + ip.replace('.', '_') + '.txt'
    print(f"\nWriting output to {fname}")
    with open(fname, 'w') as f:
        for server in pot_stale_servers:
            f.write(f"rm server {server}\n")
    print(f"\nGenerated {fname} file.\n Copy this file at a particular path on the specified NS, and then run NS CLI command: batch -f <path>/{fname} to remove the stale servers AFTER careful examination of the same.")
    
def main(ip, user, password):
    session_id = login(ip, user, password)

    if session_id:
        try:
            headers = {
                'Content-Type': 'application/json',
                'X-NITRO-USER': user,
                'X-NITRO-PASS': password,
                'Cookie': f'NITRO_AUTH_TOKEN={session_id}'
            }
        
            runconfig = get_runningconfig(ip, headers)
        
            servers = get_server_entries(runconfig)
            print(f"\nTotal no. of Servers: {len(servers)}")
            services = get_service_entries(runconfig)
            # print(f"\nServices: {services}")
            svcgrpmembers = get_servicegroupmember_entries(runconfig)
            # print(f"\nServicegroup members: {svcgrpmembers}")

            pot_stale_servers = servers - services - svcgrpmembers
            print(f"\nTotal no. of Potential Stale Servers: {len(pot_stale_servers)}")

            generate_rm_commands(pot_stale_servers, ip)

        finally:
            #logout(ip, session_id)
            pass

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="NetScaler Nitro Script to list down POTENTIAL stale server(IP based) entries")
    parser.add_argument("ip", help="NetScaler Management IP address")
    parser.add_argument("user", help="NetScaler username")
    parser.add_argument("password", help="NetScaler password")
    args = parser.parse_args()
    main(args.ip, args.user, args.password)
