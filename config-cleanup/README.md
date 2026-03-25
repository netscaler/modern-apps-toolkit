# nsic_cleanup.py

Clean up stale NSIC (NetScaler Ingress Controller) configuration from a NetScaler by name prefix.

## Problem

When NSIC is deleted without first removing the associated Ingress/Gateway resources, it leaves behind stale configuration on the NetScaler. Since other config may exist on the same appliance, a blanket `clear config` is not safe.

## Solution

This script deletes only the resources whose names start with a given prefix (e.g. `k8s-` or `k8s_`), covering all entity types that NSIC creates.

## Usage

```bash
# Dry run — preview what would be deleted (no changes made):
python nsic_cleanup.py --nsip <NSIP> --user <USER> --password <PASS> --prefix <PREFIX> --dry-run

# Actually delete:
python nsic_cleanup.py --nsip <NSIP> --user <USER> --password <PASS> --prefix <PREFIX>

# Also delete .crt/.key files from /nsconfig/ssl/:
python nsic_cleanup.py --nsip <NSIP> --user <USER> --password <PASS> --prefix <PREFIX> --delete-certkeyfiles
```

## Options

| Flag | Description |
|------|-------------|
| `--nsip` | NetScaler management IP or FQDN |
| `--user` | NetScaler username |
| `--password` | NetScaler password |
| `--prefix` | Name prefix to match (e.g. `k8s`). Auto-expands to `k8s-` and `k8s_` |
| `--dry-run` | List what would be deleted without making changes |
| `--delete-certkeyfiles` | Also delete SSL files from `/nsconfig/ssl/` |
| `--debug` | Enable verbose debug logging |

## What Gets Deleted

- **Vservers**: csvserver, lbvserver, gslbvserver
- **Service groups**: servicegroup, gslbservicegroup
- **Monitors**: lbmonitor
- **Policies & actions**: cspolicy, csaction, responderpolicy, responderaction, rewritepolicy, rewriteaction, sslpolicy, sslaction, appfwpolicy, botpolicy, authenticationpolicy, etc.
- **Profiles**: nshttpprofile, nstcpprofile, sslprofile, analyticsprofile, appfwprofile, botprofile, etc.
- **Policy data structures**: policydataset, policypatset, policystringmap, nsvariable
- **SSL cert keys**: sslcertkey, sslcertkeybundle, sslcacertbundle
- **Other**: ipset, server, nspbr (PBR), nslimitidentifier, streamselector, sslcipher

**Excluded**: IP routes (static routes).

## Notes

- Failed deletions are retried up to 4 times to handle ordering issues
- SSL cert key files in `/nsconfig/ssl/` are only deleted with `--delete-certkeyfiles`

