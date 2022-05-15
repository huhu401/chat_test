package svr

import (
	"github.com/huhu401/chat_test/chat/player"
	"github.com/huhu401/chat_test/constant"
	"github.com/huhu401/chat_test/msg"
	"log"
	"net"
	"sync"
)

var wt = sync.WaitGroup{}

func StartServe(port int) {
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IP{0, 0, 0, 0}, Port: port})
	if err != nil {
		log.Fatalln("监听端口失败", port, "error", err)
		return
	}
	defer ln.Close()
	// Acceptor.
	wt.Add(1)
	go accept(ln)
	wt.Wait()
}

func accept(ln *net.TCPListener) {
	defer wt.Done()
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Fatalln("accept error", err)
		}
		wt.Add(1)
		go svr(c)
	}
}

func svr(c net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("svr recover err", r)
		}
		c.Close()
		wt.Done()
	}()
	var p *player.Player
	for {
		m, err := msg.Read(c, true)
		if err != nil {
			return
		}
		if m.Grp == constant.MsgGrpLogin && m.Cmd == constant.MsgCmdLogin {
			p = player.GetManager().Login(m.Data.(*msg.ReqMsgLogin))
			p.C = c
			p.ASyncExec(p.Resp, uint8(constant.MsgGrpLogin), uint8(constant.MsgCmdLogin), &msg.RspMsgLogin{Status: 0})
		} else {
			if p != nil {
				p.Cast(m)
			}
		}
	}
}
