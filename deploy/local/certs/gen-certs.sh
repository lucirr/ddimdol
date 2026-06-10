#!/usr/bin/env bash
# gen-certs.sh — Generate mTLS certificates for local development
# Usage: ./gen-certs.sh [--force] [<edge-id>]
#   --force   Overwrite existing certificate files
#   edge-id   Identifier for the client certificate (default: edge-local-01)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="$SCRIPT_DIR"

FORCE=false
EDGE_ID="edge-local-01"

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --force)
            FORCE=true
            ;;
        --*)
            echo "Unknown option: $arg" >&2
            exit 1
            ;;
        *)
            EDGE_ID="$arg"
            ;;
    esac
done

echo "==> Output directory : $OUT_DIR"
echo "==> Edge ID          : $EDGE_ID"
echo "==> Force overwrite  : $FORCE"
echo ""

# Helper: return 0 (true) if the file needs to be generated.
# If --force, remove the file so callers always recreate it.
need_generate() {
    local file="$1"
    if [[ "$FORCE" == "true" ]]; then
        rm -f "$file"
        return 0
    fi
    [[ ! -f "$file" ]]
}

# ── 1. CA ─────────────────────────────────────────────────────────────────────

CA_KEY="$OUT_DIR/ca.key"
CA_CRT="$OUT_DIR/ca.crt"

if need_generate "$CA_KEY" || need_generate "$CA_CRT"; then
    rm -f "$CA_KEY" "$CA_CRT"  # 한 쪽만 없을 때 쌍 맞추기
    echo "==> Generating CA key and certificate (10 years)..."
    # WARNING: no passphrase — local development only. Use HSM or encrypted key for production CA.
    openssl genrsa -out "$CA_KEY" 4096
    openssl req -new -x509 -days 3650 -key "$CA_KEY" -out "$CA_CRT" \
        -subj "/CN=EdgeDIP CA/O=EdgeDIP/C=KR"
    echo "  created: $CA_KEY, $CA_CRT"
else
    echo "  skipped: $CA_KEY, $CA_CRT (use --force to regenerate)"
fi

# ── 2. Server certificate ─────────────────────────────────────────────────────

SERVER_KEY="$OUT_DIR/server.key"
SERVER_CSR="$OUT_DIR/server.csr"
SERVER_CRT="$OUT_DIR/server.crt"
SERVER_EXT="$OUT_DIR/.server-ext.cnf"

if need_generate "$SERVER_KEY"; then
    echo "==> Generating server key and certificate (2 years)..."

    cat > "$SERVER_EXT" <<EOF
[req]
req_extensions = v3_req
distinguished_name = dn
[dn]
[v3_req]
subjectAltName = @alt_names
extendedKeyUsage = serverAuth
[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
EOF

    openssl genrsa -out "$SERVER_KEY" 2048 2>/dev/null
    openssl req -new \
        -key "$SERVER_KEY" \
        -out "$SERVER_CSR" \
        -subj "/C=KR/ST=Seoul/O=Local Dev/CN=localhost" \
        -config "$SERVER_EXT"
    openssl x509 -req \
        -in "$SERVER_CSR" \
        -CA "$CA_CRT" \
        -CAkey "$CA_KEY" \
        -CAcreateserial \
        -out "$SERVER_CRT" \
        -days 730 \
        -sha256 \
        -extfile "$SERVER_EXT" \
        -extensions v3_req
    rm -f "$SERVER_CSR" "$SERVER_EXT"
    echo "  [created] $SERVER_KEY"
    echo "  [created] $SERVER_CRT"
fi

# ── 3. Client certificate ─────────────────────────────────────────────────────

CLIENT_KEY="$OUT_DIR/client-${EDGE_ID}.key"
CLIENT_CSR="$OUT_DIR/client-${EDGE_ID}.csr"
CLIENT_CRT="$OUT_DIR/client-${EDGE_ID}.crt"
CLIENT_EXT="$OUT_DIR/.client-ext.cnf"

if need_generate "$CLIENT_KEY"; then
    echo "==> Generating client key and certificate for '$EDGE_ID' (2 years)..."

    cat > "$CLIENT_EXT" <<EOF
[req]
req_extensions = v3_req
distinguished_name = dn
[dn]
[v3_req]
extendedKeyUsage = clientAuth
EOF

    openssl genrsa -out "$CLIENT_KEY" 2048 2>/dev/null
    openssl req -new \
        -key "$CLIENT_KEY" \
        -out "$CLIENT_CSR" \
        -subj "/C=KR/ST=Seoul/O=Local Dev/CN=${EDGE_ID}" \
        -config "$CLIENT_EXT"
    openssl x509 -req \
        -in "$CLIENT_CSR" \
        -CA "$CA_CRT" \
        -CAkey "$CA_KEY" \
        -CAcreateserial \
        -out "$CLIENT_CRT" \
        -days 730 \
        -sha256 \
        -extfile "$CLIENT_EXT" \
        -extensions v3_req
    rm -f "$CLIENT_CSR" "$CLIENT_EXT"
    echo "  [created] $CLIENT_KEY"
    echo "  [created] $CLIENT_CRT"
fi

# ── Summary ───────────────────────────────────────────────────────────────────

echo ""
echo "==> Certificate files in $OUT_DIR :"
for f in "$CA_KEY" "$CA_CRT" "$SERVER_KEY" "$SERVER_CRT" "$CLIENT_KEY" "$CLIENT_CRT"; do
    [[ -f "$f" ]] && echo "    $f"
done
echo ""
echo "Done."
