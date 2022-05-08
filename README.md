# TOOL

English | [简体中文](README-CN.md)

## Overview

* Build HTTP and GRPC services
* Quickly build SQL conditional statements
* The LRU cache
* Filters include Bloom and Cuckoo
* The unique ID can be GUID, NanoID, Snowflake, or UUID
* The Websocket
* Parameter validator
* Type conversion
* Verify the characters、AES、DES、RSA
* Prometheus monitoring and link tracing
* A distributed lock
* The log level

## SQL Where 
* conditional statements are built quickly

```go
import (
  "github.com/wjoj/tool"
)

type Account struct{
    Name `json:"name" gorm:"column:name" ifs:"="`
}

whs := NewWhereStructureFuncIfs(&Account{
    Name: "wjoj",
} , "gorm column", func(key string) string {
   if key == "name" {
       return "="
   }
   return "="
})

wh := new(tool.Where)
wh.AndIf("phone","like", "%28%")
wh.AndWhereStructure(whs, "or")
```