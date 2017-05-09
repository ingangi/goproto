package main

import (
	"fmt"
	"protoserver"
	"os"
	. "typedefs"
)

func main() {
	defer fmt.Println("goproto over")
	fmt.Println("goproto start")

	ProtoServer := new(protoserver.ProtoServer)
	if nil != ProtoServer.Listen(ServerConfig.ServerIP, ServerConfig.ServerPort) {
		fmt.Println("exiting on listen error")
		os.Exit(1)
	}

	ProtoServer.Run()
}
