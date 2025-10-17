

# Patient Privacy – Fabric + Vault + MinIO (MVP)

A minimal, production-lean MVP for patient‑controlled health‑record sharing:
- **Hyperledger Fabric** → consent state, break‑glass, immutable audit
- **MinIO (S3)** → encrypted bundle storage (off‑chain)
- **Vault** → key custody (transit + KV) for DEK/KEK/PRE/ABE roots
- **Keycloak** → identity (doctors, HOD, attributes)
- **(Optional) Postgres** → orchestrator idempotency/bookmarks

This README shows **exact steps** to initialize and start the project on a fresh machine, plus quick tests and troubleshooting.

---

## 0) Prerequisites

- OS: Linux/macOS (Windows → WSL2)
- Installed:
  - Docker **Desktop** (macOS) or Docker Engine (Linux) + Compose plugin
  - `git`, `curl`, `jq`, `make`, `netcat` (`nc`)
  - Go ≥ **1.21** (to build chaincode)
  - Hyperledger Fabric samples (downloaded in step 6)

Verify:
```bash
docker version
docker compose version
jq --version
```

---

## 1) Project layout

```bash
mkdir -p ~/patient-privacy/{compose,scripts,chaincode}
cd ~/patient-privacy
```

Result:
```
patient-privacy/
  compose/
  scripts/
  chaincode/
  README.rm
```

---

## 2) Environment variables

Create `compose/.env`:
```env
# Vault
VAULT_DEV_ROOT_TOKEN_ID=root
VAULT_ADDR=http://vault:8200

# MinIO
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=adminadmin
MINIO_BUCKET=ehr

# Keycloak
KEYCLOAK_USER=admin
KEYCLOAK_PASSWORD=admin

# Orchestrator DB (optional)
POSTGRES_USER=orchestrator
POSTGRES_PASSWORD=orchestratorpass
POSTGRES_DB=orchestrator
```

---

## 3) Docker Compose stack (Phase 0)

Create `compose/docker-compose.yml`:
```yaml
version: "3.9"

services:
  vault:
    image: hashicorp/vault:1.17
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: ${VAULT_DEV_ROOT_TOKEN_ID}
    command: server -dev -dev-root-token-id=${VAULT_DEV_ROOT_TOKEN_ID}
    ports: ["8200:8200"]
    cap_add: ["IPC_LOCK"]
    restart: unless-stopped

  minio:
    image: minio/minio:RELEASE.2025-02-15T00-00-00Z
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    command: server /data --console-address ":9001"
    ports: ["9000:9000","9001:9001"]
    volumes:
      - minio-data:/data
    restart: unless-stopped

  keycloak:
    image: quay.io/keycloak/keycloak:25.0
    command: start-dev
    environment:
      KEYCLOAK_ADMIN: ${KEYCLOAK_USER}
      KEYCLOAK_ADMIN_PASSWORD: ${KEYCLOAK_PASSWORD}
    ports: ["8080:8080"]
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports: ["5432:5432"]
    volumes:
      - pg-data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  minio-data:
  pg-data:
```

Start the stack:
```bash
cd ~/patient-privacy/compose
docker compose up -d
docker compose ps
```

---

## 4) Initialize Vault (inside container)

Create `scripts/init_vault.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

VAULT_CID=$(docker ps -qf "name=vault")
[ -z "$VAULT_CID" ] && { echo "Vault not running"; exit 1; }

docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root "$VAULT_CID" sh -lc '
set -e
vault secrets enable -path=transit transit || true
vault secrets enable -path=kv kv-v2 || true
vault write -f transit/keys/kek-root type=aes256-gcm96
vault write -f transit/keys/hpke-root type=aes256-gcm96
vault write -f transit/keys/pre-root type=aes256-gcm96
vault write -f transit/keys/abe-root type=aes256-gcm96
vault list transit/keys || true
'
curl -s http://localhost:8200/v1/sys/health | jq .
```

Run it:
```bash
bash ~/patient-privacy/scripts/init_vault.sh
```

---

## 5) Initialize MinIO (bucket + versioning + retention)

Create `scripts/init_minio.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
MINIO_CID=$(docker ps -qf "name=minio")
[ -z "$MINIO_CID" ] && { echo "MinIO not running"; exit 1; }

docker exec "$MINIO_CID" sh -lc '
  set -e
  apk add --no-cache mc >/dev/null
  mc alias set local http://localhost:9000 '"${MINIO_ROOT_USER:-admin}"' '"${MINIO_ROOT_PASSWORD:-adminadmin}"'
  mc mb -p local/'"${MINIO_BUCKET:-ehr}"' || true
  mc version enable local/'"${MINIO_BUCKET:-ehr}"'
  mc ilm add --id retain-7y --expiry-days 2555 local/'"${MINIO_BUCKET:-ehr}"' || true
  echo "✅ MinIO bucket ready"
'
```

Run:
```bash
bash ~/patient-privacy/scripts/init_minio.sh
```

Check:
```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:9000/minio/health/ready
# expect 200
```
Console: http://localhost:9001 (admin/adminadmin)

---

## 6) Hyperledger Fabric test network

Install Fabric binaries and config (once):
```bash
cd ~
git clone https://github.com/hyperledger/fabric-samples.git
cd fabric-samples
curl -sSL https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh | bash -s -- binary
export PATH=$HOME/fabric-samples/bin:$PATH
export FABRIC_CFG_PATH=$HOME/fabric-samples/config
```

Start network + channel `consent`:
```bash
cd ~/fabric-samples/test-network
./network.sh down
./network.sh up createChannel -c consent -ca
```

Sanity:
```bash
. ./scripts/envVar.sh
setGlobals 1
peer channel list
```

---

## 7) Chaincode scaffold (consentcc)

Create folder:
```bash
mkdir -p ~/patient-privacy/chaincode/consentcc
cd ~/patient-privacy/chaincode/consentcc
go mod init consentcc && go mod tidy
```

Create `chaincode.go`:
```go
package main

import (
  "encoding/json"
  "fmt"
  "time"
  "github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct{ contractapi.Contract }

type Bundle struct {
  ID, Owner, Type, URI, SHA256, ArchetypeID, CreatedAt string
  Size int64
}

type Consent struct {
  BundleID, DoctorID, Scope, Purpose, ExpiresAt, Status string
}

func (s *SmartContract) CreateBundle(ctx contractapi.TransactionContextInterface,
        id, owner, btype, uri, sha string, size int64, arch string) error {
  b := Bundle{ID: id, Owner: owner, Type: btype, URI: uri, SHA256: sha, Size: size, ArchetypeID: arch,
    CreatedAt: time.Now().UTC().Format(time.RFC3339)}
  by, _ := json.Marshal(b)
  return ctx.GetStub().PutState("BNDL_"+id, by)
}

func (s *SmartContract) GrantConsent(ctx contractapi.TransactionContextInterface,
        bundleId, doctorId, scope, purpose, expiresAt string) error {
  c := Consent{BundleID: bundleId, DoctorID: doctorId, Scope: scope, Purpose: purpose,
    ExpiresAt: expiresAt, Status: "GRANTED"}
  by, _ := json.Marshal(c)
  return ctx.GetStub().PutState(fmt.Sprintf("CONS_%s_%s", bundleId, doctorId), by)
}

func (s *SmartContract) RevokeConsent(ctx contractapi.TransactionContextInterface,
        bundleId, doctorId string) error {
  key := fmt.Sprintf("CONS_%s_%s", bundleId, doctorId)
  st, err := ctx.GetStub().GetState(key)
  if err != nil || st == nil { return fmt.Errorf("no consent found") }
  var c Consent; _ = json.Unmarshal(st, &c)
  c.Status = "REVOKED"
  by, _ := json.Marshal(c)
  return ctx.GetStub().PutState(key, by)
}

func main() {
  cc, err := contractapi.NewChaincode(new(SmartContract))
  if err != nil { panic(err) }
  if err := cc.Start(); err != nil { panic(err) }
}
```

Deploy:
```bash
cd ~/fabric-samples/test-network
./network.sh deployCC -ccn consentcc -ccp ~/patient-privacy/chaincode/consentcc -ccl go -c consent
```

---

## 8) Minimal CLI test

```bash
cd ~/fabric-samples/test-network
. ./scripts/envVar.sh
setGlobals 1

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
 --tls --cafile "$ORDERER_CA" -C consent -n consentcc \
 --peerAddresses localhost:7051 --tlsRootCertFiles "$PEER0_ORG1_CA" \
 -c '{"Args":["CreateBundle","bndl1","patient1","allergy","s3://ehr/bndl1.enc","sha123","1024","ISO13606:AllergyIntolerance"]}'

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
 --tls --cafile "$ORDERER_CA" -C consent -n consentcc \
 --peerAddresses localhost:7051 --tlsRootCertFiles "$PEER0_ORG1_CA" \
 -c '{"Args":["GrantConsent","bndl1","doctor1","read","treatment","2025-12-31T00:00:00Z"]}'

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
 --tls --cafile "$ORDERER_CA" -C consent -n consentcc \
 --peerAddresses localhost:7051 --tlsRootCertFiles "$PEER0_ORG1_CA" \
 -c '{"Args":["RevokeConsent","bndl1","doctor1"]}'
```

---

## 9) Optional: Hyperledger Explorer (read‑only UI)

- Clone: `git clone https://github.com/hyperledger-labs/blockchain-explorer`
- Mount Fabric crypto (`organizations/`) into Explorer at `/tmp/crypto`
- Ensure Explorer joins the same Docker network as your peers
- Open UI → http://localhost:8081 (port mapping per your compose)

> In the connection profile, use **container hostnames** (`peer0.org1.example.com`, `orderer.example.com`) and point `tlsCACerts.path` to `/tmp/crypto/.../tls/ca.crt`. Set `DISCOVERY_AS_LOCALHOST=false`.

---

## 10) Smoke test script

Create `scripts/smoke.sh`:
```bash
#!/usr/bin/env bash
set -e
echo "Vault:";     curl -s http://localhost:8200/v1/sys/health | jq '.initialized'
echo "MinIO:";     curl -s -o /dev/null -w "%{http_code}\n" http://localhost:9000/minio/health/ready
echo "Keycloak:";  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/
echo "Postgres:";  (nc -z localhost 5432 && echo ok) || echo skip
echo "Fabric peers:"; docker ps | grep peer0.org1.example.com && echo OK || echo FAIL
```

---

## 11) Makefile shortcuts

Create `Makefile`:
```make
up:
	cd compose && docker compose up -d

down:
	cd compose && docker compose down

init:
	bash scripts/init_vault.sh
	bash scripts/init_minio.sh

smoke:
	bash scripts/smoke.sh
```

Run:
```bash
make up
make init
make smoke
```

---

## Troubleshooting

- **`peer: Peer binary and configuration files not found`**  
  Install Fabric binaries and set:
  ```bash
  export PATH=$HOME/fabric-samples/bin:$PATH
  export FABRIC_CFG_PATH=$HOME/fabric-samples/config
  ```

- **`Failed to parse channel configuration, install jq`**  
  `sudo apt-get install -y jq` or `brew install jq`

- **`Cannot init crypto: .../config/msp not found`**  
  Use helper envs:
  ```bash
  . ./scripts/envVar.sh && setGlobals 1
  ```
  Or set `CORE_PEER_MSPCONFIGPATH` to Org Admin MSP:
  `.../organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp`

- **MinIO `mc: not found`**  
  Our script installs `mc` inside the container each run.

- **Explorer “Default client peer is down”**  
  Put Explorer on the same Docker network, use `grpcs://peer0.org1.example.com:7051`, mount `/tmp/crypto`, set `DISCOVERY_AS_LOCALHOST=false`.

---

## Next (Phase 2)

- Event‑driven **orchestrator** (Go): subscribe to Fabric events and trigger PRE/ABE key ops via Vault.
- **Admin Dashboard** (Next.js): HOD approvals, consent queries, audit export.
- Add **Emergency** chaincode functions (Request/Approve/Grant/Revoke) with TTL.

---

**Contact / Notes**

- Dev mode credentials are for local only. Change passwords and apply TLS/mTLS in real deployments.
- All PHI must be encrypted client‑side before upload to MinIO.
# 🧩 ConsentChain – Patient-Controlled Health Record Sharing (Fabric + Vault + MinIO)

This project implements a **patient-centric, privacy-preserving health data-sharing platform**.  
It combines **Hyperledger Fabric**, **HashiCorp Vault**, and **MinIO** to allow patients to **grant, revoke, and audit access** to encrypted health record bundles.

---

## ⚙️ Stack Overview

| Component | Purpose |
|------------|----------|
| **Fabric** | Blockchain ledger for consent, revocation, and audit trail |
| **Vault** | Manages encryption keys (transit + KV engines) |
| **MinIO** | Stores encrypted off-chain bundles |
| **Keycloak** | Identity provider for doctors, departments (HOD) |
| **Postgres** | Orchestrator state / idempotency (optional) |

---

## 🪜 1. Prerequisites

| Tool | Version |
|------|----------|
| Docker / Docker Desktop | ≥ 24.x |
| Docker Compose Plugin | ≥ 2.x |
| Git, curl, jq, make, nc | latest |
| Go | ≥ 1.21 |
| Hyperledger Fabric Samples | installed later |

Verify:
```bash
docker version
docker compose version
jq --version
```

---

## 🏗️ 2. Folder Structure

```bash
mkdir -p ~/consentchain/{compose,scripts,chaincode}
cd ~/consentchain
```

```
consentchain/
 ├─ compose/
 ├─ scripts/
 ├─ chaincode/
 └─ README.md
```

---

## 🧾 3. Environment File

Create `compose/.env`:
```env
VAULT_DEV_ROOT_TOKEN_ID=root
VAULT_ADDR=http://vault:8200

MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=adminadmin
MINIO_BUCKET=ehr

KEYCLOAK_USER=admin
KEYCLOAK_PASSWORD=admin

POSTGRES_USER=orchestrator
POSTGRES_PASSWORD=orchestratorpass
POSTGRES_DB=orchestrator
```

---

## 🐳 4. Docker Compose – Core Services

`compose/docker-compose.yml`
```yaml
version: "3.9"

services:
  vault:
    image: hashicorp/vault:1.17
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: ${VAULT_DEV_ROOT_TOKEN_ID}
    command: server -dev -dev-root-token-id=${VAULT_DEV_ROOT_TOKEN_ID}
    ports: ["8200:8200"]
    cap_add: ["IPC_LOCK"]
    restart: unless-stopped

  minio:
    image: minio/minio:RELEASE.2025-02-15T00-00-00Z
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    command: server /data --console-address ":9001"
    ports: ["9000:9000","9001:9001"]
    volumes:
      - minio-data:/data
    restart: unless-stopped

  keycloak:
    image: quay.io/keycloak/keycloak:25.0
    command: start-dev
    environment:
      KEYCLOAK_ADMIN: ${KEYCLOAK_USER}
      KEYCLOAK_ADMIN_PASSWORD: ${KEYCLOAK_PASSWORD}
    ports: ["8080:8080"]
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports: ["5432:5432"]
    volumes:
      - pg-data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  minio-data:
  pg-data:
```

Launch:
```bash
cd compose
docker compose up -d
docker compose ps
```

---

## 🔐 5. Initialize Vault

`scripts/init_vault.sh`
```bash
#!/usr/bin/env bash
set -euo pipefail
VAULT_CID=$(docker ps -qf "name=vault")
[ -z "$VAULT_CID" ] && { echo "Vault not running"; exit 1; }

docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root "$VAULT_CID" sh -lc '
vault secrets enable -path=transit transit || true
vault secrets enable -path=kv kv-v2 || true
for k in kek-root hpke-root pre-root abe-root; do
  vault write -f transit/keys/$k type=aes256-gcm96
done
vault list transit/keys
'
```
Run:
```bash
bash scripts/init_vault.sh
```

---

## 🪣 6. Initialize MinIO

`scripts/init_minio.sh`
```bash
#!/usr/bin/env bash
set -euo pipefail
CID=$(docker ps -qf "name=minio")
[ -z "$CID" ] && { echo "MinIO not running"; exit 1; }

docker exec "$CID" sh -lc '
apk add --no-cache mc >/dev/null
mc alias set local http://localhost:9000 admin adminadmin
mc mb -p local/ehr || true
mc version enable local/ehr
mc ilm add --id retain-7y --expiry-days 2555 local/ehr || true
echo "✅ MinIO bucket ready"
'
```
Run & check:
```bash
bash scripts/init_minio.sh
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:9000/minio/health/ready
```
UI → [http://localhost:9001](http://localhost:9001) (`admin/adminadmin`)

---

## 🧩 7. Fabric Network Setup

```bash
cd ~
git clone https://github.com/hyperledger/fabric-samples.git
cd fabric-samples
curl -sSL https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh | bash -s -- binary
export PATH=$HOME/fabric-samples/bin:$PATH
export FABRIC_CFG_PATH=$HOME/fabric-samples/config
```

Start network + channel:
```bash
cd ~/fabric-samples/test-network
./network.sh down
./network.sh up createChannel -c consent -ca
```

Verify peer context:
```bash
. ./scripts/envVar.sh
setGlobals 1
peer channel list   # expect "consent"
```

---

## 🪶 8. Deploy Chaincode `consentcc`

```bash
mkdir -p ./patient-privacy/chaincode/
cd ./patient-privacy/chaincode/
go mod init consentcc && go mod tidy
```

`chaincode.go`
```go
package main
// (use your consentcc chaincode implementation here)
```

### Package

Format
```
peer lifecycle chaincode package <chain_name>.tar.gz --path <path_to_main_go> --lang golang --label <label_name>
```
Example:
```
peer lifecycle chaincode package consentcc.tar.gz --path ./cmd --lang golang --label consentgo_1
```

### Deploy:
```bash
cd ~/fabric-samples/test-network
./network.sh deployCC -ccn consentcc -ccp ~/consentchain/chaincode/consentcc -ccl go -c consent
```

---

## 🧪 9. Test on CLI

```bash
cd ~/fabric-samples/test-network
. ./scripts/envVar.sh
setGlobals 1

peer chaincode invoke -o localhost:7050 \
 --ordererTLSHostnameOverride orderer.example.com \
 --tls --cafile "$ORDERER_CA" \
 -C consent -n consentcc \
 --peerAddresses localhost:7051 \
 --tlsRootCertFiles "$PEER0_ORG1_CA" \
 -c '{"Args":["CreateBundle","bndl1","patient1","allergy","s3://ehr/bndl1.enc","sha123","1024","ISO13606:AllergyIntolerance"]}'

peer chaincode invoke -C consent -n consentcc \
 -c '{"Args":["GrantConsent","bndl1","doctor1","read","treatment","2025-12-31T00:00:00Z"]}'

peer chaincode invoke -C consent -n consentcc \
 -c '{"Args":["RevokeConsent","bndl1","doctor1"]}'
```

---

## 🌐 10. Optional – Hyperledger Explorer (UI)

```bash
git clone https://github.com/hyperledger-labs/blockchain-explorer
cd blockchain-explorer
```

- Copy Fabric crypto:  
  `cp -r ~/fabric-samples/test-network/organizations ./artifacts/`
- Mount in compose:  
  `- ./artifacts/organizations:/tmp/crypto:ro`
- Join same Docker network as Fabric (`external: true`)
- Edit connection profile → use container hostnames (`peer0.org1.example.com`, `orderer.example.com`)
- Set `DISCOVERY_AS_LOCALHOST=false`

Launch:
```bash
docker compose up -d
```
Visit: **http://localhost:8081**

---

## 🧾 11. Smoke Test Script

`scripts/smoke.sh`
```bash
#!/usr/bin/env bash
set -e
echo "Vault:"; curl -s http://localhost:8200/v1/sys/health | jq '.initialized'
echo "MinIO:"; curl -s -o /dev/null -w "%{http_code}\n" http://localhost:9000/minio/health/ready
echo "Keycloak:"; curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/
echo "Postgres:"; (nc -z localhost 5432 && echo ok) || echo skip
echo "Fabric peers:"; docker ps | grep peer0.org1.example.com && echo OK || echo FAIL
```

Run:
```bash
bash scripts/smoke.sh
```

---

## 🧰 12. Makefile (shortcuts)

```make
up:
	cd compose && docker compose up -d
down:
	cd compose && docker compose down
init:
	bash scripts/init_vault.sh
	bash scripts/init_minio.sh
smoke:
	bash scripts/smoke.sh
```

---

## ⚠️ Troubleshooting

| Issue | Fix |
|-------|-----|
| **Peer binary not found** | `export PATH=$HOME/fabric-samples/bin:$PATH` |
| **jq missing** | `sudo apt install jq` or `brew install jq` |
| **Cannot init crypto (config/msp)** | `. ./scripts/envVar.sh && setGlobals 1` |
| **MinIO “mc not found”** | Script installs it inside container automatically |
| **Explorer peer down** | Ensure same Docker network + correct TLS paths + `DISCOVERY_AS_LOCALHOST=false` |
| **Port conflicts** | Adjust mappings in compose (`8080→8081`, etc.) |

---

## 🧭 Next Phase

- Go orchestrator listening to Fabric events (grant/revoke/TTL).
- Next.js admin dashboard for approvals & audit.
- Emergency flow (break-glass with HOD approval).
- Add ABE/PRE key services (Vault-backed).

---

### ✅ Summary Commands

```bash
make up        # start Vault/MinIO/Keycloak/Postgres
make init      # initialize Vault + MinIO
cd ~/fabric-samples/test-network && ./network.sh up createChannel -c consent -ca
./network.sh deployCC -ccn consentcc -ccp ~/consentchain/chaincode/consentcc -ccl go -c consent
bash ~/consentchain/scripts/smoke.sh
```

---

> _ConsentChain – a secure, auditable, patient-centric consent ledger for healthcare data._