package main

import (
	"flag"
	"fmt"
	"github.com/huhu401/chat_test/chat/player"
	"github.com/huhu401/chat_test/chat/svr"
	"github.com/huhu401/chat_test/gen_routine"
	"github.com/huhu401/chat_test/profanity"
)

func init() {
	beforeMain()
}

type flagArgs struct {
	Port int
}

var Args = flagArgs{}

func beforeMain() {
	flag.IntVar(&Args.Port, "p", 8888, "指定监听端口")
	flag.Parse()
	gen_routine.BeforeMain()
	player.BeforeMain()
	profanity.BeforeMain()
}

func main() {
	fmt.Println("i am chat")
	svr.StartServe(Args.Port)
}
