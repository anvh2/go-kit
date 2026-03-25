# Resource Upload Architecture

## Overview
There are two upload paths depending on the caller.

---

## Path 1: Client Upload (via Cloudflare Worker)

Used by end-user clients (e.g., web/app) uploading audio, images, etc.

```
User
 │
 │  POST {gw-domain}/resource/private/upload
 ▼
Cloudflare Edge
 │
 ├─ [CF Worker intercepts route pattern]
 │
 ├─ [1] Authenticate via Kong:
 │      {gw-domain}/auth/internal/verify-token
 │
 ├─ [2] Upload file bytes directly to Cloudflare R2
 │      (same CF network — no GCP egress cost)
 │
 └─ [3] Save metadata to backend:
         {gw-domain}/resource/private/resources/metadata/save
              │
              ▼
           Kong Gateway
              │
              ▼  (strips /resource/private → /private)
           Go Backend (resource)
              │
              ▼
           MySQL (resource record)
```

### Why CF Worker?

- File bytes never touch GCP — no egress cost  
- R2 upload is near-zero latency (same CF network)  
- Backend only handles lightweight metadata JSON  

---

## Path 2: Internal/Backend Upload (via Kong → Go service)

Used by internal services (gRPC `InternalUpload`, `InternalUploadV2`).

```
Internal Service (gRPC)
 │
 ▼
Kong Gateway
 │  /resource/private/* → /private/*
 ▼
Go Backend (resource)
 │  handler: /private/resources/upload
 ▼
Cloudflare R2 (via PutObject)
 │
 ▼
MySQL (resource record)
```

### Notes

- The backend HTTP upload handler (`/private/resources/upload`) includes:
  - Concurrent upload limiting via Redis counters  
  - Protects pod memory under high load  

---

## Kong Route Mapping (QA)

| External Path                              | Internal Path                     | Handler                |
| ------------------------------------------ | --------------------------------- | ---------------------- |
| /resource/private/*                        | /private/*                        | Go backend (generic)   |
| /resource/private/resources/placement-test | /private/resources/placement-test | Placement Test service |
| /resource/internal/*                       | /internal/*                       | Go backend (internal)  |
| /resource/private/upload                   | (CF Worker intercepts)            | Cloudflare Worker → R2 |

---

## Key Components

| Component                             | Role                                                                            |
| ------------------------------------- | ------------------------------------------------------------------------------- |
| Cloudflare Worker (cf-gamehub-worker) | Edge upload handler, direct R2 write                                            |
| Cloudflare R2                         | Object storage (tech-dev-generic-resources, tech-dev-generic-private-resources) |
| Kong Gateway                          | Auth, path rewriting, rate limiting                                             |
| resource (Go)                         | Metadata persistence, internal upload API                                       |
| MySQL                                 | Resource records                                                                |
| Redis                                 | Concurrent upload counters, presigned URL cache                                 |
