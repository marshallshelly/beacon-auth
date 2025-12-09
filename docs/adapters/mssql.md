---
title: MSSQL Adapter
description: Using BeaconAuth with Microsoft SQL Server
---

BeaconAuth provides a native adapter for Microsoft SQL Server using `github.com/microsoft/go-mssqldb`.

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
go get github.com/microsoft/go-mssqldb
```

## Usage

```go
package main

import (
    "context"
    "log"

    "github.com/marshallshelly/beacon-auth/beaconauth"
    "github.com/marshallshelly/beacon-auth/adapters/mssql"
)

func main() {
    ctx := context.Background()

    // Initialize adapter
    adapter, err := mssql.New(ctx, &mssql.Config{
        Host:     "localhost",
        Port:     1433,
        Database: "myapp",
        Username: "sa",
        Password: "StrongPassword123",
        Params: map[string]string{
            "encrypt": "disable", // Set to false for local dev usually
        },
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

## Features

- Uses `OUTPUT Inserted.*` for efficient returns.
- Supports `OFFSET`/`FETCH` pagination (SQL Server 2012+).
