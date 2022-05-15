package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/huhu401/chat_test/constant"
	"github.com/huhu401/chat_test/msg"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func init() {
	beforeMain()
}

type flagArgs struct {
	Port int
}

type client struct {
	roleId     int64
	name       string
	logicChan  chan interface{}
	connection net.Conn
	ctx        context.Context
	ctxCancel  context.CancelFunc
}

var Args = flagArgs{}
var Client *client

func beforeMain() {
	flag.IntVar(&Args.Port, "p", 8888, "指定服务器端口")
	flag.Parse()
}

func main() {
	fmt.Println("i am client")
	b := bufio.NewReader(os.Stdin)
	for {
		line, _, err := b.ReadLine()
		if err != nil {
			log.Println("scan error", err)
			return
		}
		err = handleCmd(string(line))
		if err != nil {
			log.Println("exit main loop", err)
			break
		}
	}
}

func handleCmd(in string) error {
	sl := strings.Split(in, " ")
	switch sl[0] {
	case "login":
		roleId, _ := strconv.ParseInt(sl[1], 0, 64)
		err := startClient(roleId, sl[2])
		if err != nil {
			return err
		}
	case "chat":
		chat(sl[1])
	case "join":
		i, _ := strconv.Atoi(sl[1])
		join(i)
	case "exit":
		return errors.New("manual exit")
	default:
		log.Println("unknown cmd ", sl)
	}
	return nil
}

func chat(content string) {
	if Client == nil {
		log.Println("还没有登录，请先登录")
		return
	}
	Client.logicChan <- content
}

func join(grpId int) {
	if Client == nil {
		log.Println("还没有登录，请先登录")
		return
	}
	Client.logicChan <- grpId
}

func startClient(roleId int64, name string) error {
	Client = &client{roleId: roleId, name: name, logicChan: make(chan interface{})}
	Client.ctx, Client.ctxCancel = context.WithCancel(context.Background())
	con, err := connect()
	if err != nil {
		return err
	}
	Client.connection = con
	go Client.netLoop()
	go Client.logicLoop()
	return nil
}

func connect() (net.Conn, error) {
	c, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(Args.Port))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *client) login() error {
	m := &msg.ReqMsgLogin{RoleId: c.roleId}
	_, err := msg.Write(c.connection, &msg.Message{Grp: constant.MsgGrpLogin, Cmd: constant.MsgCmdLogin, Data: m})
	return err
}

func (c *client) join(grpId int) {
	m := &msg.ReqMsgJoin{Grp: int32(grpId)}
	_, err := msg.Write(c.connection, &msg.Message{Grp: constant.MsgGrpChat, Cmd: constant.MsgCmdJoin, Data: m})
	if err != nil {
		log.Println("write join msg fail", err)
	}
}

func (c *client) chat(content string) {
	d := &msg.ReqMsgChat{Content: content}
	_, err := msg.Write(c.connection, &msg.Message{Grp: constant.MsgGrpChat, Cmd: constant.MsgCmdChat, Data: d})
	if err != nil {
		log.Println("write join msg fail", err)
	}
}

func doClose() {
	Client.ctxCancel()
	err := Client.connection.Close()
	log.Println("terminate", err)
}

func (c *client) netLoop() {
	defer doClose()
	for {
		m, err := msg.Read(c.connection, false)
		if err != nil {
			log.Println("connection read fail", err)
			return
		}
		c.logicChan <- m
	}
}

func (c *client) logicLoop() {
	defer doClose()
	err := c.login()
	if err != nil {
		log.Println("send login fail", err)
		return
	}
	for {
		select {
		case <-c.ctx.Done():
			break
		case m := <-c.logicChan: // 处理发进来的消息
			err = c.handle(m)
			// 返回错误，则退出
			if err != nil {
				break
			}
		}
	}
}

func (c *client) handle(m interface{}) error {
	log.Println("received msg ", m)
	switch v := m.(type) {
	case string:
		c.chat(v)
	case *msg.Message:
		return c.handleSvrMsg(v)
	case int:
		c.join(v)
	}
	return nil
}

func (c *client) handleSvrMsg(m *msg.Message) error {
	switch m.Grp {
	case constant.MsgGrpLogin:
		c.handleLogin(m)
	case constant.MsgGrpChat:
		c.handleChat(m)
	default:
		log.Println("received unknown msg", m)
	}
	return nil
}

func (c *client) handleLogin(m *msg.Message) {
	switch m.Cmd {
	case constant.MsgCmdLogin:
		log.Println("received login ret msg", m.Data.(*msg.RspMsgLogin))
	}
}

func (c *client) handleChat(m *msg.Message) {
	switch m.Cmd {
	case constant.MsgCmdChat:
		log.Println("received chat ret msg", m.Data.(*msg.RspMsgChat))
	case constant.MsgCmdJoin:
		log.Println("received join ret msg", m.Data.(*msg.RspMsgJoin))
	case constant.MsgCmdNotify:
		log.Println("received new chat msg", m.Data.(*msg.RspMsgNotify))
	case constant.MsgCmdHistory:
		log.Println("received chat history back msg", m.Data.(*msg.RspMsgHistory))
	}
}
