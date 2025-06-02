#!/usr/bin/env bash
set -euo pipefail

# -----------------------------------------------------------------------------
# Script to deploy a 3-AZ TiDB cluster with labeled TiKV and TiDB nodes via TiUP
# -----------------------------------------------------------------------------

# Configuration (override via env vars if needed)
CLUSTER_NAME="${CLUSTER_NAME:-stale-read-demo}"
TIDB_VERSION="${TIDB_VERSION:-v8.5.1}"
TOPOLOGY_FILE="topology-3az.yaml"

tiup cluster destroy ${CLUSTER_NAME} -y || true

# Check connectivity and port availability before deploying
#echo "Checking connectivity and port availability..."
#tiup cluster check ${TOPOLOGY_FILE}

tiup cluster deploy ${CLUSTER_NAME} ${TIDB_VERSION} -y ./${TOPOLOGY_FILE}

tiup cluster start ${CLUSTER_NAME}

# Configure TiDB to read from closest replicas
mysql -h 10.148.0.15 -P 4000 -u root -e "SET GLOBAL tidb_replica_read='closest-replicas';"
