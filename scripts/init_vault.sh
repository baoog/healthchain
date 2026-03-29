#!/usr/bin/env bash
set -euo pipefail

# --- Basic Vault env (dev setup) ---
export VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
export VAULT_TOKEN="${VAULT_TOKEN:-${VAULT_DEV_ROOT_TOKEN_ID:-root}}"
export HOSPITAL_ID="${HOSPITAL_ID:-hospital-a}"

echo "[*] Using VAULT_ADDR=${VAULT_ADDR}"

# --- Enable transit + kv if not already enabled ---
vault secrets enable -path=transit transit 2>/dev/null || echo "[*] transit already enabled"
vault secrets enable -path=federation kv-v2 2>/dev/null      || echo "[*] kv-v2 already enabled"

# --- Store hospital IDs in Vault ---
vault kv put secret/data/federation/hospital-b id="$HOSPITAL_ID"

# --- NEW: Ed25519 transit keys for signing ---

# Global bundle-signing key (or one per hospital if bạn muốn tách theo domain)
echo "[*] Creating Ed25519 transit key for bundle signing..."
vault write -f transit/keys/"$HOSPITAL_ID"-bundle-key type="ed25519" >/dev/null 2>&1 || true

# Example: per-hospital signing keys (if you want separate domains)
echo "[*] Creating Ed25519 transit keys per hospital (optional)..."
vault write -f transit/keys/"$HOSPITAL_ID"-payload-key type="ed25519" >/dev/null 2>&1 || true

# --- List keys to verify ---
echo "[*] Current transit keys:"
vault list transit/keys || true