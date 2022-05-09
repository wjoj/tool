package util

import (
	"fmt"
	"log"
	"runtime/debug"
)

func GoSave(fn func()) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[ERROR]: %s", fmt.Sprintf("%s\n%s", p, string(debug.Stack())))
		}
	}()

	fn()
}
