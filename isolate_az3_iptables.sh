#!/usr/bin/env bash
set -euo pipefail

# Usage: $0 [block|unblock]
ACTION="${1:-block}"

# AZ1 and AZ2 hosts to isolate
REMOTE_HOSTS=(10.148.0.15 10.148.0.16)

for host in "${REMOTE_HOSTS[@]}"; do
  if [ "$ACTION" = "block" ]; then
    iptables -I OUTPUT -d "$host" -j DROP
    iptables -I INPUT  -s "$host" -j DROP
    echo "Blocked all traffic to/from $host"
  else
    iptables -D OUTPUT -d "$host" -j DROP
    iptables -D INPUT  -s "$host" -j DROP
    echo "Unblocked all traffic to/from $host"
  fi
done

echo "Current iptables rules:"
iptables -L --line-numbers
