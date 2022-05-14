# TOOL

English | [简体中文](README-CN.md)

## Overview

* Build HTTP and GRPC services
* Quickly build SQL conditional statements
* The LRU cache、local cache
* Filters include Bloom and Cuckoo
* The unique ID can be GUID, NanoID, Snowflake, or UUID
* The Websocket、socket
* Parameter validator
* Type conversion
* Verify the characters、AES、DES、RSA
* Prometheus monitoring and link tracing
* A distributed lock
* The log level
* Current limiter

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

## Socket

```go
type Message struct {
    Lng uint32
    I   uint16
    Msg string
}
```

Server

```go
SocketListen(899, func(s *SocketConn) error {
    readMsg := new(Message)
    err := s.ReadBody().Numerical(&readMsg.Lng)
    if err != nil {
        t.Errorf("\nReadBody error  lng:%+v	err:%v", readMsg.Lng, err)
        return err
    }
    err = s.ReadBody().Numerical(&readMsg.I)
    if err != nil {
        t.Errorf("\nReadBody error  ri:%v	err:%v", readMsg.I, err)
        return err
    }
    body, err := s.ReadBody().Body(int64(readMsg.Lng))
    if err != nil {
        t.Errorf("\nBody error  ri:%v	err:%v", len(body), err)
        return err
    }
    readMsg.Msg = string(body)
    t.Logf("\nread body succ msg:%+v", readMsg)
    msgb := fmt.Sprintf("server:%+v kkkkkkk", readMsg.I)
    writeMsg := &Message{
        Msg: msgb,
        Lng: uint32(len(msgb)),
        I:   readMsg.I,
    }

    msg := NewBodyWrite()
    msg.Numerical(&writeMsg.Lng)
    msg.Numerical(&writeMsg.I)
    msg.Write([]byte(writeMsg.Msg))
    _, err = s.WriteBody(msg)
    if err != nil {
        t.Errorf("\nWriteBody error i:%+v err:%v", writeMsg.I, err)
        return err
    }
    t.Logf("\nwrite body succ:%+v", writeMsg.I)
    // time.Sleep(time.Second * 2)
    return nil
})
```

Client

```go
for i := uint16(0); i < uint16(t.N); i++ {
    // fmt.Printf("\ni:%+v", i)
    j := i
    SocketClient("127.0.0.1:899", func(s *SocketConn) error {
        msgb := fmt.Sprintf("client:%+v", i)
        writeMsg := &Message{
            Msg: msgb,
            Lng: uint32(len(msgb)),
            I:   j,
        }
        msg := NewBodyWrite()
        err := msg.Numerical(writeMsg.Lng)
        if err != nil {
            // t.Fatalf("WriteBody error  lng i:%+v	err:%v", j, err)
            return err
        }
        err = msg.Numerical(&writeMsg.I)
        if err != nil {
            // t.Fatalf("\nWriteBody error  i i:%+v	err:%v", j, err)
            return err
        }
        _, err = msg.Write([]byte(writeMsg.Msg))
        if err != nil {
            // t.Fatalf("\nWriteBody error  body i:%+v	err:%v", j, err)
            return err
        }
        _, err = s.WriteBody(msg)
        if err != nil {
            // t.Fatalf("\nWriteBody error i:%+v	err:%v", j, err)
            return err
        }
        // t.Logf("\nWriteBody succ i:%+v	", j)
        readMsg := new(Message)
        err = s.ReadBody().Numerical(&readMsg.Lng)
        if err != nil {
            // t.Fatalf("\nReadBody error i:%v lng:%+v err:%v", j, readMsg.Lng, err)
            return err
        }
        err = s.ReadBody().Numerical(&readMsg.I)
        if err != nil {
            // t.Fatalf("\nReadBody error i:%+v  ri:%v	err:%v", j, readMsg.I, err)
            return err
        }
        body, err := s.ReadBody().Body(int64(readMsg.Lng))
        if err != nil {
            // t.Fatalf("\nBody error i:%+v  ri:%v	err:%v", j, len(body), err)
            return err
        }
        readMsg.Msg = string(body)
        // t.Logf("\nread body i:%v msg:%+v", j, readMsg)
        time.Sleep(time.Second * 2)
        return nil
    })
}
```