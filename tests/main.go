package main

import "github.com/wjoj/tool/v2"

func main() {
	tool.NewApp().
		Redis().
		Gorm().
		HttpServer().
		Run()
}
