#!/bin/sh
set -eu

usage() {
    echo "Usage: sudo ./install.sh --endpoint https://HOST:8443 --ca PATH [--token TOKEN] [--name NAME]"
    exit 2
}

endpoint=""
ca_file=""
token=""
device_name="$(hostname)"

while [ "$#" -gt 0 ]; do
    case "$1" in
        --endpoint) [ "$#" -ge 2 ] || usage; endpoint="$2"; shift 2 ;;
        --ca) [ "$#" -ge 2 ] || usage; ca_file="$2"; shift 2 ;;
        --token) [ "$#" -ge 2 ] || usage; token="$2"; shift 2 ;;
        --name) [ "$#" -ge 2 ] || usage; device_name="$2"; shift 2 ;;
        -h|--help) usage ;;
        *) echo "Unknown argument: $1" >&2; usage ;;
    esac
done

[ "$(id -u)" -eq 0 ] || { echo "Run this installer with sudo." >&2; exit 1; }
[ -n "$endpoint" ] || usage
[ -f "$ca_file" ] || { echo "CA file not found: $ca_file" >&2; exit 1; }
case "$endpoint" in https://*) ;; *) echo "Endpoint must use https://" >&2; exit 1 ;; esac

bundle_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
for file in edge-agent edge-agentctl config.yaml edge-agent.service; do
    [ -f "$bundle_dir/$file" ] || { echo "Bundle is missing $file" >&2; exit 1; }
done

"$bundle_dir/edge-agentctl" version >/dev/null

if command -v systemctl >/dev/null 2>&1; then
    systemctl stop edge-agent.service 2>/dev/null || true
fi

install -d -m 0750 /etc/edge-agent
install -d -m 0700 /var/lib/edge-agent /var/lib/edge-agent/identity
install -m 0755 "$bundle_dir/edge-agent" /usr/bin/edge-agent
install -m 0755 "$bundle_dir/edge-agentctl" /usr/bin/edge-agentctl
install -m 0644 "$ca_file" /etc/edge-agent/ca.pem

if [ ! -f /etc/edge-agent/config.yaml ]; then
    install -m 0640 "$bundle_dir/config.yaml" /etc/edge-agent/config.yaml
fi
sed -i \
    -e "s|^  endpoint:.*|  endpoint: $endpoint|" \
    -e "s|^  ca_file:.*|  ca_file: /etc/edge-agent/ca.pem|" \
    /etc/edge-agent/config.yaml

if command -v systemctl >/dev/null 2>&1; then
    install -m 0644 "$bundle_dir/edge-agent.service" /etc/systemd/system/edge-agent.service
    systemctl daemon-reload
fi

if [ -n "$token" ]; then
    /usr/bin/edge-agentctl -config /etc/edge-agent/config.yaml queue archive
    /usr/bin/edge-agentctl -config /etc/edge-agent/config.yaml enroll -token "$token" -name "$device_name"
fi

if command -v systemctl >/dev/null 2>&1; then
    systemctl enable --now edge-agent.service
fi

echo "Soul Room agent installation complete."
if [ -z "$token" ]; then
    if /usr/bin/edge-agentctl -config /etc/edge-agent/config.yaml identity show >/dev/null 2>&1; then
        echo "Existing device enrollment preserved."
        /usr/bin/edge-agentctl -config /etc/edge-agent/config.yaml status || true
    else
        echo "Enroll it with: sudo edge-agentctl -config /etc/edge-agent/config.yaml enroll -token TOKEN -name '$device_name'"
    fi
else
    /usr/bin/edge-agentctl -config /etc/edge-agent/config.yaml status || true
fi
