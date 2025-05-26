package main

import "github.com/wjoj/tool"

func main() {
	tool.NewApp().
		Redis().
		Gorm().
		HttpServer().
		Run()
}
