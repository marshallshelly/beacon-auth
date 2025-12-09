---
title: MySQL Adapter
description: Using BeaconAuth with MySQL
---

BeaconAuth provides a native adapter for MySQL databases.

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
go get github.com/go-sql-driver/mysql
```

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/marshallshelly/beacon-auth/beaconauth"
    "github.com/marshallshelly/beacon-auth/adapters/mysql"
)

func main() {
    ctx := context.Background()

    // Initialize adapter
    adapter, err := mysql.New(ctx, &mysql.Config{
        Host:     "localhost",
        Port:     3306,
        Database: "myapp",
        Username: "root",
        Password: "password",
        // Optional parameters
        MaxConns: 20,
        MinConns: 5,
        Params: map[string]string{
            "charset": "utf8mb4",
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

## Helper Functions

The adapter supports standard CRUD operations and transactions.

### Transaction Example

```go
err := adapter.Transaction(ctx, func(tx core.Adapter) error {
    // Operations inside transaction
    _, err := tx.Create(ctx, "users", data)
    return err
})
```
