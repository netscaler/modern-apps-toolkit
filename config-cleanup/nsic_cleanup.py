#!/usr/bin/env python3
"""
nsic_cleanup.py - Delete stale NSIC-created NetScaler resources by name prefix.

Requires Python 3.7 or newer.

Usage:
    python nsic_cleanup.py --nsip <NSIP> --user <USERNAME> --password <PASSWORD> --prefix <PREFIX>
    python nsic_cleanup.py --nsip <NSIP> --user <USERNAME> --password <PASSWORD> --prefix <PREFIX> --dry-run

When NSIC (NetScaler Ingress Controller) is deleted, it may leave behind stale
configuration on the NetScaler. This script finds and deletes all resources
whose names start with the given prefix.

Notes:
    - IP routes (iproute) are excluded: they have no name prefix in NSIC.
    - SSL cert keys (sslcertkey, sslcertkeybundle, sslcacertbundle) are deleted
      by default along with all other NSIC-created resources.
    - SSL files (.crt/.key) matching the prefix are removed from
      /nsconfig/ssl/ on the NetScaler only when --delete-certkeyfiles is passed.
    - The deletion order is chosen to satisfy NetScaler dependency rules:
      policies & actions first, then vservers, service groups, monitors, then
      standalone objects (profiles, datasets, cipher groups, etc.).
"""

from __future__ import annotations

import argparse
import logging
import re
import requests
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s  %(levelname)-8s  %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger("nsic_cleanup")

# ---------------------------------------------------------------------------
# Resource catalogue
# Each entry describes one NetScaler resource type:
#   resource   - NITRO URL path component
#   name_field - attribute that holds the resource name / key
#   extra_del  - callable(obj) -> dict of extra query-string params for DELETE
#                (e.g. lbmonitor requires ?type=...)
# ---------------------------------------------------------------------------

# Resources are listed in the order they should be deleted (leaves first so
# that parent resources have no live bindings when we try to remove them).
RESOURCE_TYPES = [
    # ---- policies / actions (must come before vservers) --------------------
    {"resource": "cspolicy",                    "name_field": "policyname"},
    {"resource": "csaction",                    "name_field": "name"},
    {"resource": "responderpolicy",             "name_field": "name"},
    {"resource": "responderaction",             "name_field": "name"},
    {"resource": "rewritepolicy",               "name_field": "name"},
    {"resource": "rewriteaction",               "name_field": "name"},
    # sslpolicy/sslaction: legacy SSL policies NSIC created before sslprofile
    {"resource": "sslpolicy",                   "name_field": "name"},
    {"resource": "sslaction",                   "name_field": "name"},
    {"resource": "authorizationpolicy",         "name_field": "name"},
    {"resource": "authenticationpolicy",        "name_field": "name"},
    {"resource": "authenticationloginschemapolicy", "name_field": "name"},
    {"resource": "authenticationloginschema",   "name_field": "name"},
    {"resource": "authenticationoauthaction",   "name_field": "name"},
    {"resource": "authenticationsamlaction",    "name_field": "name"},
    {"resource": "authenticationldapaction",    "name_field": "name"},
    {"resource": "authenticationvserver",       "name_field": "name"},
    {"resource": "appfwpolicy",                 "name_field": "name"},
    {"resource": "appfwprofile",                "name_field": "name"},
    # appfw supporting files (NSIC uploads these with prefix-based names)
    {"resource": "appfwsignatures",             "name_field": "name"},
    {"resource": "appfwhtmlerrorpage",           "name_field": "name"},
    {"resource": "appfwxmlerrorpage",            "name_field": "name"},
    {"resource": "appfwjsonerrorpage",           "name_field": "name"},
    {"resource": "appqoepolicy",                "name_field": "name"},
    {"resource": "appqoeaction",                "name_field": "name"},
    # bot: label → policy → profile (outer-to-inner so bindings are gone first)
    {"resource": "botpolicylabel",              "name_field": "labelname"},
    {"resource": "botpolicy",                   "name_field": "name"},
    {"resource": "botprofile",                  "name_field": "name"},
    {"resource": "contentinspectionpolicy",     "name_field": "name"},
    {"resource": "contentinspectionaction",     "name_field": "name"},
    {"resource": "auditmessageaction",          "name_field": "name"},
    {"resource": "policyhttpcallout",           "name_field": "name"},
    {"resource": "nsvariable",                  "name_field": "name"},
    # ---- vservers ----------------------------------------------------------
    {"resource": "gslbvserver",                 "name_field": "name"},
    {"resource": "csvserver",                   "name_field": "name"},
    {"resource": "lbvserver",                   "name_field": "name"},
    # ---- service groups ----------------------------------------------------
    {"resource": "gslbservicegroup",  "name_field": "servicegroupname"},
    {"resource": "servicegroup",      "name_field": "servicegroupname"},
    # ---- monitors ----------------------------------------------------------
    {"resource": "lbmonitor",
     "name_field": "monitorname",
     "extra_del": lambda obj: {"args": f"type:{obj.get('type', 'USER')}"}},
    # ---- profiles ----------------------------------------------------------
    {"resource": "analyticsprofile",            "name_field": "name"},
    {"resource": "nshttpprofile",               "name_field": "name"},
    {"resource": "nstcpprofile",                "name_field": "name"},
    {"resource": "sslprofile",                  "name_field": "name"},
    {"resource": "nsicapprofile",               "name_field": "name"},
    # ---- policy datasets / maps --------------------------------------------
    {"resource": "policydataset",               "name_field": "name"},
    {"resource": "policypatset",                "name_field": "name"},
    {"resource": "policystringmap",             "name_field": "name"},
    # ---- rate-limiting / stream --------------------------------------------
    {"resource": "nslimitidentifier",           "name_field": "limitidentifier"},
    {"resource": "streamselector",              "name_field": "name"},
    # ---- SSL cipher groups -------------------------------------------------
    {"resource": "sslcipher",                   "name_field": "ciphergroupname"},
    # ---- IP sets -----------------------------------------------------------
    {"resource": "ipset",                       "name_field": "name"},
    # ---- PBR (policy-based routing) ----------------------------------------
    {"resource": "nspbr",                       "name_field": "name"},
    # ---- servers -----------------------------------------------------------
    {"resource": "server",            "name_field": "name"},
    # ---- TM session params (created for auth CRD traffic management) ------
    {"resource": "tmsessionparameter",          "name_field": "name"},
    # ---- SSL cert keys (NSIC-created, safe to delete by prefix) -----------
    {"resource": "sslcertkey",                  "name_field": "certkey"},
    {"resource": "sslcertkeybundle",            "name_field": "certkeybundlename"},
    {"resource": "sslcacertbundle",             "name_field": "cacertbundlename"},
]


# ---------------------------------------------------------------------------
# Regex pattern helper
# ---------------------------------------------------------------------------

def build_nitro_regex(prefix: str) -> str:
    """
    Build a NITRO-compatible regex pattern for server-side filtering.

    If the user supplied a trailing separator ('-' or '_'), use only that
    separator in the regex. Otherwise, use [-_] to match both.

    The prefix is escaped to prevent regex metacharacter injection.

    Examples:
        'k8s'  -> /^k8s[-_]/   (matches k8s-* and k8s_*)
        'k8s-' -> /^k8s-/      (matches only k8s-*)
        'k8s_' -> /^k8s_/      (matches only k8s_*)
    """
    if prefix.endswith("-"):
        base = prefix[:-1]
        separator = "-"
    elif prefix.endswith("_"):
        base = prefix[:-1]
        separator = "_"
    else:
        base = prefix
        separator = "[-_]"

    escaped_base = re.escape(base)
    return f"/^{escaped_base}{separator}/"


# ---------------------------------------------------------------------------
# NITRO session helpers
# ---------------------------------------------------------------------------

class NitroSession:
    """Thin NITRO REST client (no SDK dependency)."""

    def __init__(self, nsip: str, username: str, password: str,
                 ssl_verify: bool = False):
        self.base = f"https://{nsip}/nitro/v1"
        self.session = requests.Session()
        self.session.verify = ssl_verify
        self.session.headers.update({"Content-Type": "application/json"})
        self._login(username, password)

    def _login(self, username: str, password: str) -> None:
        url = f"{self.base}/config/login"
        payload = {"login": {"username": username, "password": password,
                              "timeout": 1800}}
        r = self.session.post(url, json=payload, timeout=30)
        r.raise_for_status()
        data = r.json()
        if data.get("errorcode", 0) not in (0, 444):
            raise RuntimeError(f"Login failed: {data.get('message')}")
        # NITRO returns a cookie; session handles it automatically.
        log.info("Logged in to NetScaler at %s", self.base)

    def logout(self) -> None:
        try:
            url = f"{self.base}/config/logout"
            self.session.post(url, json={"logout": {}}, timeout=10)
        except Exception:
            pass
        log.info("Logged out")

    def get_ssl_files(self) -> list:
        """Return list of file objects in /nsconfig/ssl/."""
        url = f"{self.base}/config/systemfile?args=filelocation:%2Fnsconfig%2Fssl"
        try:
            r = self.session.get(url, timeout=30)
        except requests.RequestException as e:
            log.warning("GET systemfile (ssl) failed: %s", e)
            return []
        if r.status_code == 404:
            return []
        if not r.ok:
            log.debug("GET systemfile returned %s", r.status_code)
            return []
        data = r.json()
        if data.get("errorcode", 0) not in (0,):
            return []
        return data.get("systemfile", [])

    def delete_ssl_file(self, filename: str) -> bool:
        """Delete a file from /nsconfig/ssl/."""
        url = (f"{self.base}/config/systemfile/"
               f"{requests.utils.quote(filename, safe='')}"
               f"?args=filelocation:%2Fnsconfig%2Fssl")
        try:
            r = self.session.delete(url, timeout=30)
        except requests.RequestException as e:
            log.warning("DELETE systemfile/%s failed: %s", filename, e)
            return False
        if r.status_code in (200, 201):
            return True
        data = {}
        try:
            data = r.json()
        except Exception:
            pass
        if data.get("errorcode", 0) == 258:  # already absent
            return True
        log.warning("DELETE systemfile/%s returned %s: %s",
                    filename, r.status_code, data.get("message", r.text[:200]))
        return False

    def get_all(self, resource: str) -> list:
        """Return the list of all instances of a resource type."""
        url = f"{self.base}/config/{resource}"
        try:
            r = self.session.get(url, timeout=30)
        except requests.RequestException as e:
            log.warning("GET %s failed: %s", resource, e)
            return []

        if r.status_code == 404:
            # Resource type not available on this NS build/license
            return []
        if not r.ok:
            log.warning("GET %s returned %s: %s", resource, r.status_code,
                        r.text[:200])
            return []

        data = r.json()
        if data.get("errorcode", 0) not in (0,):
            # e.g. 258 = "No such resource" (empty list)
            return []
        return data.get(resource, [])

    def get_filtered(self, resource: str, name_field: str,
                     prefix: str) -> list | None:
        """
        Return instances of a resource type whose name_field matches the
        prefix, using NITRO server-side regex filtering.

        If prefix ends with '-' or '_', matches only that separator.
        Otherwise, matches both '-' and '_' separators.

        This is more efficient than get_all() + client-side filtering because
        the NetScaler filters before sending the response.

        Returns None if server-side filtering is not supported (caller should
        fall back to get_all + client-side filtering).
        """
        regex_pattern = build_nitro_regex(prefix)
        # URL-encode the filter value (but keep the filter= key literal)
        filter_val = requests.utils.quote(f"{name_field}:{regex_pattern}", safe='')
        url = f"{self.base}/config/{resource}?filter={filter_val}"
        try:
            r = self.session.get(url, timeout=30)
        except requests.RequestException as e:
            log.warning("GET %s (filtered) failed: %s", resource, e)
            return None  # Fall back to get_all

        if r.status_code == 404:
            # Resource type not available on this NS build/license
            return []
        if not r.ok:
            # Server-side filtering not supported - fall back to get_all
            log.debug("GET %s (filtered) returned %s - falling back to get_all",
                      resource, r.status_code)
            return None

        data = r.json()
        if data.get("errorcode", 0) not in (0,):
            # e.g. 258 = "No such resource" (empty list)
            return []

        return data.get(resource, [])

    def get_batch(self, resource_types: list,
                  prefix: str) -> dict | None:
        """
        Fetch multiple resource types in a SINGLE request using NITRO batch API
        with query_params for filtering.

        Args:
            resource_types: List of resource info dicts (with 'resource' and 'name_field')
            prefix: User-supplied prefix (e.g. 'k8s', 'k8s-', or 'k8s_')

        Returns:
            Dict mapping resource name -> list of matching objects, or None if batch
            API is not supported (caller should fall back to individual GETs).
        """
        # Build batch payload with query_params for filter
        # Format: {"batchapi": [{"type": "csvserver", "query_params": {"filter": {name_field: regex}}}, ...]}
        batch_items = []
        regex_pattern = build_nitro_regex(prefix)
        for rinfo in resource_types:
            resource = rinfo["resource"]
            name_field = rinfo["name_field"]
            batch_items.append({
                "type": resource,
                "query_params": {"filter": {name_field: regex_pattern}}
            })
        payload = {"batchapi": batch_items}

        url = f"{self.base}/config/batchapi?action=GET"
        headers = {"Content-Type": "application/json"}
        try:
            r = self.session.post(url, json=payload, headers=headers, timeout=120)
        except requests.RequestException as e:
            log.warning("Batch GET failed: %s", e)
            return None  # Fall back to individual GETs

        if not r.ok:
            log.debug("Batch GET returned %s: %s - falling back to individual GETs",
                      r.status_code, r.text[:300])
            return None

        try:
            results = r.json()
        except Exception:
            log.warning("Batch GET - invalid JSON response")
            return None

        # Response format: {'errorcode': 0, 'batchapi': [{resource: [...]}, ...]}
        if not isinstance(results, dict):
            log.debug("Batch GET - unexpected response format (not a dict): %s",
                      str(results)[:200])
            return None

        # Check top-level error
        if results.get("errorcode", 0) not in (0,):
            log.debug("Batch GET failed: %s", results.get("message", "Unknown"))
            return None

        # Extract the batchapi array
        batch_results = results.get("batchapi", [])
        if not isinstance(batch_results, list):
            log.debug("Batch GET - 'batchapi' is not a list: %s",
                      str(batch_results)[:200])
            return None

        # Parse results into a dict
        result_map = {}
        for i, rinfo in enumerate(resource_types):
            resource = rinfo["resource"]
            if i >= len(batch_results):
                result_map[resource] = []
                continue

            item = batch_results[i]
            if not isinstance(item, dict):
                result_map[resource] = []
                continue

            # Check for per-item error
            errorcode = item.get("errorcode", 0)
            if errorcode not in (0, 258):  # 258 = empty/not found
                result_map[resource] = []
                continue

            # Extract the resource list
            result_map[resource] = item.get(resource, [])

        log.info("Batch GET completed: fetched %d resource types in 1 request",
                 len(resource_types))
        return result_map

    def delete(self, resource: str, name: str,
               extra_params: dict | None = None,
               log_errors: bool = True) -> bool:
        """Delete a single named resource. Returns True on success."""
        url = f"{self.base}/config/{resource}/{requests.utils.quote(name, safe='')}"
        # NITRO requires colons/commas inside ?args= to be literal (not percent-encoded).
        # Build the query string manually instead of using requests params=.
        if extra_params:
            qs = "&".join(f"{k}={v}" for k, v in extra_params.items())
            url = f"{url}?{qs}"
        try:
            r = self.session.delete(url, timeout=30)
        except requests.RequestException as e:
            if log_errors:
                log.error("DELETE %s/%s failed: %s", resource, name, e)
            else:
                log.debug("DELETE %s/%s failed (will retry): %s", resource, name, e)
            return False

        if r.status_code in (200, 201):
            return True

        data = {}
        try:
            data = r.json()
        except Exception:
            pass

        errorcode = data.get("errorcode", r.status_code)
        # 258 = object does not exist (already gone)
        if errorcode == 258:
            log.debug("DELETE %s/%s: already absent", resource, name)
            return True

        if log_errors:
            log.error("DELETE %s/%s failed [%s]: %s",
                      resource, name, r.status_code,
                      data.get("message", r.text[:200]))
        else:
            log.debug("DELETE %s/%s failed [%s] (will retry): %s",
                      resource, name, r.status_code,
                      data.get("message", r.text[:200]))
        return False

    def delete_batch_all(self, items: list,
                          log_errors: bool = True,
                          show_summary: bool = True) -> tuple[list, list]:
        """
        Delete multiple resources of different types in a SINGLE NITRO request
        using the batch API with X-NITRO-ONERROR: continue.

        Args:
            items: List of (resource, name_field, name, rinfo, obj) tuples
            log_errors: Whether to log errors (False during retries)
            show_summary: Whether to display tabular summary (False during retries)

        Returns:
            (succeeded, failed) - tuple of lists of (rinfo, obj) tuples
        """
        if not items:
            return [], []

        # Build batch API payload
        # Format: {"batchapi": [{"action": "REMOVE", "type": "csvserver", "properties": {"name": "..."}}, ...]}
        batch_items = []
        for resource, name_field, name, rinfo, obj in items:
            batch_items.append({
                "action": "REMOVE",
                "type": resource,
                "properties": {name_field: name}
            })
        payload = {"batchapi": batch_items}

        url = f"{self.base}/config/batchapi"
        headers = {"X-NITRO-ONERROR": "continue"}  # Process all even if some fail
        try:
            r = self.session.post(url, json=payload, headers=headers, timeout=120)
        except requests.RequestException as e:
            if log_errors:
                log.error("BATCH DELETE failed: %s", e)
            # All failed
            return [], [(rinfo, obj) for _, _, _, rinfo, obj in items]

        # Parse response
        succeeded = []
        failed = []

        try:
            results = r.json()
        except Exception:
            # Can't parse response - treat all as failed
            if log_errors:
                log.error("BATCH DELETE failed - invalid response: %s", r.text[:200])
            return [], [(rinfo, obj) for _, _, _, rinfo, obj in items]

        # Response format: {'errorcode': 0, 'batchapi': [{errorcode, message}, ...]}
        if not isinstance(results, dict):
            if log_errors:
                log.error("BATCH DELETE failed - unexpected format: %s", str(results)[:200])
            return [], [(rinfo, obj) for _, _, _, rinfo, obj in items]

        # Note: top-level errorcode 1243 means "Bulk operation failed" which happens
        # when SOME items fail (e.g., due to dependencies). We still need to check
        # per-item results to see which succeeded vs failed.
        # Don't return early on top-level error - always check batchapi array.

        # Extract per-item results from batchapi array
        batch_results = results.get("batchapi", [])
        if not isinstance(batch_results, list):
            # Fallback: maybe all succeeded
            if log_errors:
                log.warning("BATCH DELETE - no per-item results, assuming success")
            return [(rinfo, obj) for _, _, _, rinfo, obj in items], []

        # Process per-item results and collect stats per resource type
        delete_stats = {}  # resource -> {"deleted": count, "failed": count}

        for i, (resource, name_field, name, rinfo, obj) in enumerate(items):
            if resource not in delete_stats:
                delete_stats[resource] = {"deleted": 0, "failed": 0}

            if i >= len(batch_results):
                # No result for this item - treat as failed
                failed.append((rinfo, obj))
                delete_stats[resource]["failed"] += 1
                continue

            result = batch_results[i]
            errorcode = result.get("errorcode", 0) if isinstance(result, dict) else 0

            # 0 = success, 258 = already gone (treat as success)
            if errorcode in (0, 258):
                succeeded.append((rinfo, obj))
                delete_stats[resource]["deleted"] += 1
                log.debug("    Deleted  %s/%s", resource, name)
            else:
                failed.append((rinfo, obj))
                delete_stats[resource]["failed"] += 1
                if log_errors:
                    msg = result.get("message", "Unknown error") if isinstance(result, dict) else str(result)
                    log.debug("    Failed   %s/%s [%s]: %s", resource, name, errorcode, msg)

        # Display tabular summary of batch delete results (only on final attempt)
        if delete_stats and show_summary:
            max_name_len = max(len(r) for r in delete_stats.keys())
            total_deleted = sum(s["deleted"] for s in delete_stats.values())
            total_failed = sum(s["failed"] for s in delete_stats.values())
            log.info("  %-*s  %7s  %6s", max_name_len, "Resource Type", "Deleted", "Failed")
            log.info("  %s  %s  %s", "-" * max_name_len, "-" * 7, "-" * 6)
            for resource, stats in delete_stats.items():
                log.info("  %-*s  %7d  %6d", max_name_len, resource,
                         stats["deleted"], stats["failed"])
            log.info("  %s  %s  %s", "-" * max_name_len, "-" * 7, "-" * 6)
            log.info("  %-*s  %7d  %6d", max_name_len, "TOTAL", total_deleted, total_failed)

        return succeeded, failed

    # --- Old macroapi approach (delete one resource type at a time) ---
    # def delete_batch(self, resource: str, name_field: str,
    #                  names: list, log_errors: bool = True) -> tuple[list, list]:
    #     """
    #     Delete multiple resources of the same type in a single NITRO request
    #     using the macroapi endpoint.
    #     """
    #     if not names:
    #         return [], []
    #     items = [{name_field: n} for n in names]
    #     payload = {"macroapi": {"action": "rm", resource: items}}
    #     url = f"{self.base}/config/macroapi"
    #     r = self.session.post(url, json=payload, timeout=60)
    #     # macroapi returns all-or-nothing, no per-item status


# ---------------------------------------------------------------------------
# SSL file cleanup
# ---------------------------------------------------------------------------

def delete_ssl_files(session: NitroSession, prefixes: list,
                     dry_run: bool) -> tuple[int, int]:
    """
    Delete .crt/.key/.pem files from /nsconfig/ssl/ whose names start with any
    of the given prefixes.  Returns (found, deleted) counts.
    """
    SSL_EXTENSIONS = (".crt", ".key", ".pem")
    files = session.get_ssl_files()
    matched = [
        f for f in files
        if isinstance(f, dict)
        and f.get("filename", "").endswith(SSL_EXTENSIONS)
        and any(f.get("filename", "").startswith(p) for p in prefixes)
    ]
    if not matched:
        return 0, 0

    log.info("Found %d SSL file(s) in /nsconfig/ssl/ matching prefix(es) %s",
             len(matched), "/".join(repr(p) for p in prefixes))
    deleted = 0
    for f in matched:
        fname = f.get("filename", "")
        if dry_run:
            log.info("  [DRY-RUN] Would delete /nsconfig/ssl/%s", fname)
            deleted += 1
        else:
            log.info("  Deleting /nsconfig/ssl/%s ...", fname)
            ok = session.delete_ssl_file(fname)
            if ok:
                log.info("  Deleted  /nsconfig/ssl/%s", fname)
                deleted += 1
            else:
                log.warning("  Failed to delete /nsconfig/ssl/%s", fname)
    return len(matched), deleted


# ---------------------------------------------------------------------------
# Main cleanup logic
# ---------------------------------------------------------------------------

def collect_matching(session: NitroSession, resource_info: dict,
                     prefix: str, prefixes: list) -> list:
    """
    Fetch instances of a resource type whose name_field starts with any of
    the given prefixes.  Tries server-side filtering first, falls back to
    client-side filtering if server doesn't support it.
    """
    resource = resource_info["resource"]
    name_field = resource_info["name_field"]

    # Try server-side filtering first
    server_matched = session.get_filtered(resource, name_field, prefix)
    if server_matched is not None:
        # Post-filter to match only user-specified prefixes
        return [
            obj for obj in server_matched
            if isinstance(obj, dict)
            and any(str(obj.get(name_field, "")).startswith(p) for p in prefixes)
        ]

    # Fall back to get_all + client-side filtering
    log.debug("Falling back to client-side filtering for %s", resource)
    items = session.get_all(resource)
    matched = []
    for obj in items:
        if not isinstance(obj, dict):
            continue
        name = str(obj.get(name_field, ""))
        if any(name.startswith(p) for p in prefixes):
            matched.append(obj)
    return matched


def run_cleanup(nsip: str, username: str, password: str, prefix: str,
                dry_run: bool, delete_certkeyfiles: bool = False) -> None:

    # NSIC uses both '-' and '_' as separators (e.g. k8s-lbvs and k8s_sg).
    # Strip trailing separator to get base prefix for regex filter.
    if prefix.endswith("-") or prefix.endswith("_"):
        base_prefix = prefix[:-1]
        prefixes = [prefix]
    else:
        base_prefix = prefix
        prefixes = [prefix + "-", prefix + "_"]

    # Validate prefix: NSIC only allows alphanumeric prefixes
    if not base_prefix.isalnum():
        log.error('Invalid prefix: "%s" is not alphanumeric (only a-z, A-Z, 0-9 allowed)',
                  base_prefix)
        return

    log.info("Matching prefix: %r (base: %r)", prefixes, base_prefix)

    session = NitroSession(nsip, username, password)

    try:
        total_found = 0
        total_deleted = 0

        resource_list = list(RESOURCE_TYPES)

        # pending: list of (rinfo, obj) that should be deleted
        pending = []

        # Try batch GET first (all resource types in ONE request)
        batch_result = session.get_batch(resource_list, prefix)

        # Collect matching resources and build summary for tabular display
        resource_counts = []  # [(resource_type, count), ...]

        if batch_result is not None:
            # Batch GET succeeded - post-filter to ensure exact prefix match
            for rinfo in resource_list:
                resource = rinfo["resource"]
                name_field = rinfo["name_field"]
                server_matched = batch_result.get(resource, [])
                # Post-filter: only keep items whose name starts with a user prefix
                matched = [
                    obj for obj in server_matched
                    if isinstance(obj, dict)
                    and any(str(obj.get(name_field, "")).startswith(p) for p in prefixes)
                ]
                if not matched:
                    continue
                total_found += len(matched)
                resource_counts.append((resource, len(matched)))
                for obj in matched:
                    pending.append((rinfo, obj))
        else:
            # Batch GET not supported - fall back to individual GETs
            log.info("Batch GET not supported, falling back to individual requests")
            for rinfo in resource_list:
                resource = rinfo["resource"]
                name_field = rinfo["name_field"]

                matched = collect_matching(session, rinfo, prefix, prefixes)
                if not matched:
                    continue

                total_found += len(matched)
                resource_counts.append((resource, len(matched)))
                for obj in matched:
                    pending.append((rinfo, obj))

        # Display discovery results in tabular format
        if resource_counts:
            max_name_len = max(len(r) for r, _ in resource_counts)
            log.info("")
            log.info("Resources matching prefix(es) %s:", ", ".join(repr(p) for p in prefixes))
            log.info("  %-*s  %s", max_name_len, "Resource Type", "Count")
            log.info("  %s  %s", "-" * max_name_len, "-----")
            for resource, count in resource_counts:
                log.info("  %-*s  %5d", max_name_len, resource, count)
            log.info("  %s  %s", "-" * max_name_len, "-----")
            log.info("  %-*s  %5d", max_name_len, "TOTAL", total_found)
            log.info("")

        # Delete pass with up to MAX_RETRIES retry rounds for items that fail.
        # A deletion may fail because a dependency (bound resource) was deleted
        # later in the same pass; the next round usually succeeds.
        MAX_RETRIES = 4
        for attempt in range(1, MAX_RETRIES + 2):   # rounds: 1 .. MAX_RETRIES+1
            if not pending:
                break
            still_failed = []
            is_first = (attempt == 1)
            is_last = (attempt == MAX_RETRIES + 1)
            if attempt > 1:
                log.info("")
                log.info("── Retry round %d/%d (%d item(s) remaining) ──",
                         attempt - 1, MAX_RETRIES, len(pending))

            if dry_run:
                # Dry-run: just log what would be deleted
                for rinfo, obj in pending:
                    resource = rinfo["resource"]
                    name = obj.get(rinfo["name_field"], "")
                    log.info("  [DRY-RUN] Would delete %s/%s", resource, name)
                    total_deleted += 1
                pending = []
            else:
                # Try batch delete first (all items in ONE request)
                # Separate items with extra_del (need special handling) from regular items
                batch_items = []
                individual_items = []
                for rinfo, obj in pending:
                    if rinfo.get("extra_del"):
                        # Resources like lbmonitor need extra args - do individually
                        individual_items.append((rinfo, obj))
                    else:
                        resource = rinfo["resource"]
                        name_field = rinfo["name_field"]
                        name = obj.get(name_field, "")
                        batch_items.append((resource, name_field, name, rinfo, obj))

                # Batch delete regular items
                if batch_items:
                    log.info("")
                    log.info("Batch deleting %d resource(s)...", len(batch_items))
                    succeeded, batch_failed = session.delete_batch_all(
                        batch_items, log_errors=is_last, show_summary=(is_first or is_last))
                    total_deleted += len(succeeded)
                    still_failed.extend(batch_failed)

                # Individual delete for items with extra_del (e.g., lbmonitor)
                for rinfo, obj in individual_items:
                    resource = rinfo["resource"]
                    name_field = rinfo["name_field"]
                    extra_del_fn = rinfo.get("extra_del")
                    name = obj.get(name_field, "")
                    extra = extra_del_fn(obj) if extra_del_fn else None
                    log.info("  Deleting %s/%s ...", resource, name)
                    ok = session.delete(resource, name, extra, log_errors=is_last)
                    if ok:
                        log.info("    Deleted  %s/%s", resource, name)
                        total_deleted += 1
                    else:
                        still_failed.append((rinfo, obj))

            pending = still_failed

        total_failed = len(pending)

        # Delete SSL files from /nsconfig/ssl/ matching the prefix
        if delete_certkeyfiles:
            ssl_files_found, ssl_files_deleted = delete_ssl_files(
                session, prefixes, dry_run)
            total_found   += ssl_files_found
            total_deleted += ssl_files_deleted

        # Summary
        print()
        print("=" * 60)
        if dry_run:
            print(f"DRY-RUN complete. Would delete {total_found} resource(s)/file(s).")
        else:
            print(f"Cleanup complete.")
            print(f"  Resources found   : {total_found}")
            print(f"  Deleted           : {total_deleted}")
            print(f"  Failed            : {total_failed}")
            if pending:
                print()
                print("  Failed resources (could not delete after retries):")
                for rinfo, obj in pending:
                    name = obj.get(rinfo["name_field"], "?")
                    print(f"    {rinfo['resource']}/{name}")
        print("=" * 60)

    finally:
        session.logout()


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description="Delete stale NSIC-created NetScaler config by name prefix.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Preview what would be deleted (no changes made):
  python nsic_cleanup.py --nsip 10.0.0.1 --user nsroot --password MyPass \\
      --prefix k8s_ --dry-run

  # Actually delete all resources starting with 'k8s_':
  python nsic_cleanup.py --nsip 10.0.0.1 --user nsroot --password MyPass \\
      --prefix k8s_

  # Also delete .crt/.key files from /nsconfig/ssl/:
  python nsic_cleanup.py --nsip 10.0.0.1 --user nsroot --password MyPass \\
      --prefix k8s_ --delete-certkeyfiles

Resource types examined (name-prefix-filterable, routes excluded):

  LB/CS/GSLB vservers:
    csvserver, lbvserver, gslbvserver

  Service groups:
    servicegroup, gslbservicegroup

  Monitors:
    lbmonitor

  Policies & actions:
    cspolicy, csaction,
    responderpolicy, responderaction,
    rewritepolicy, rewriteaction,
    sslpolicy, sslaction,
    appfwpolicy, appqoepolicy, appqoeaction,
    botpolicylabel, botpolicy,
    contentinspectionpolicy, contentinspectionaction,
    authorizationpolicy, authenticationpolicy,
    authenticationvserver,
    authenticationoauthaction, authenticationsamlaction, authenticationldapaction,
    authenticationloginschema, authenticationloginschemapolicy,
    auditmessageaction, policyhttpcallout

  Profiles:
    nshttpprofile, nstcpprofile, sslprofile, sslcipher,
    analyticsprofile, nsicapprofile, appfwprofile, botprofile,
    appfwsignatures, appfwhtmlerrorpage, appfwxmlerrorpage, appfwjsonerrorpage

  Policy data structures:
    policydataset, policypatset, policystringmap, nsvariable

  Rate-limiting / stream:
    nslimitidentifier, streamselector

  Network:
    ipset, server, tmsessionparameter

  SSL cert keys (always included):
    sslcertkey, sslcertkeybundle, sslcacertbundle

  Optional (--delete-certkeyfiles):
    SSL files in /nsconfig/ssl/ matching the prefix: <prefix>*.crt, <prefix>*.key
""",
    )
    p.add_argument("--nsip",     required=True, help="NetScaler management IP or FQDN")
    p.add_argument("--user",     required=True, help="NetScaler username")
    p.add_argument("--password", required=True, help="NetScaler password")
    p.add_argument("--prefix",   required=True,
                   help="Name prefix to match (e.g. 'k8s' or 'k8s_'). "
                        "If no trailing '-' or '_' is given, both 'prefix-' "
                        "and 'prefix_' are matched automatically.")
    p.add_argument("--dry-run",  action="store_true",
                   help="List what would be deleted without making any changes")
    p.add_argument("--delete-certkeyfiles", action="store_true",
                   help="Also delete the .crt/.key files from /nsconfig/ssl/ "
                        "that match the prefix (off by default)")
    p.add_argument("--debug",    action="store_true",
                   help="Enable verbose debug logging")
    return p.parse_args()


def main() -> None:
    args = parse_args()
    if args.debug:
        log.setLevel(logging.DEBUG)

    if args.dry_run:
        log.info("DRY-RUN mode: no changes will be made")

    run_cleanup(
        nsip=args.nsip,
        username=args.user,
        password=args.password,
        prefix=args.prefix,
        dry_run=args.dry_run,
        delete_certkeyfiles=args.delete_certkeyfiles,
    )


if __name__ == "__main__":
    main()