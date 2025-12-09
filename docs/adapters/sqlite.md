---
title: SQLite Adapter
description: Using BeaconAuth with SQLite
---

BeaconAuth provides a native adapter for SQLite via `modernc.org/sqlite` (pure Go implementation).

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
go get modernc.org/sqlite
```

## Usage

```go
package main

import (
    "context"
    "log"

    "github.com/marshallshelly/beacon-auth/beaconauth"
    "github.com/marshallshelly/beacon-auth/adapters/sqlite"
)

func main() {
    ctx := context.Background()

    // Initialize adapter
    adapter, err := sqlite.New(ctx, &sqlite.Config{
        DataSourceName: "file:data.db?cache=shared&mode=rwc",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer adapter.Close()

    // Initialize BeaconAuth
    auth, err := beaconauth.New(
        beaconauth.WithAdapter(adapter),
        // ...
    )
}
```

## Concurrency Note

SQLite generally allows only one writer at a time. The adapter configures connection pool settings (MaxOpenConns=1) to prevent "database is locked" errors in WAL mode scenarios without complex retries.
