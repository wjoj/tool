# TOOL

[English](README.md) | 简体中文

## 概述

* 构建HTTP、GPRC服务
* 快速生成SQL条件语句
* LRU缓存
* 过滤器有布隆、布谷
* 唯一ID生成方式： GUID, NanoID, Snowflake, or UUID
* Websocket

## SQL条件语句

* 快速生成条件语句

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