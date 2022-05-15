package player

import (
	"fmt"
	"github.com/huhu401/chat_test/constant"
	"github.com/huhu401/chat_test/gen_routine"
	"github.com/huhu401/chat_test/msg"
	"github.com/huhu401/chat_test/profanity"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// Player 玩家对象
type Player struct {
	RoleID int64 //玩家RoleId
	C      net.Conn
	*gen_routine.Svr
	chatGrp    int32 // 聊天室编号
	LoginStamp int64
}

// NewPlayer 创建一个临时玩家对象，是否成功还要看能不能跑得起协程来
// 所以这里不能写玩家确定登陆进来的逻辑
func NewPlayer(req *msg.ReqMsgLogin) *Player {
	p := &Player{RoleID: req.RoleId}
	return p
}

// Stop  协程处理 注意，只能由别的协程来通知
func (p *Player) Stop(reason *gen_routine.Error) {
	p.StopSvr(reason)
}

func (p *Player) Key() interface{} {
	return p.RoleID
}

func (p *Player) Init(svr *gen_routine.Svr) *gen_routine.Error {
	p.Svr = svr
	p.LoginStamp = time.Now().Unix()
	return nil
}

func (p *Player) alreadyIn(req *msg.ReqMsgLogin) {
	log.Println("player already in", p.LogId(), "req", req)
}

// Terminate 协程内调过来的退出操作
func (p *Player) Terminate(rea *gen_routine.Error) {
	p.GrpByeBye()
	err := p.C.Close()
	if rea.Code == gen_routine.ErrorCodeOk || rea.Code == gen_routine.ErrorGateOffline || rea.Code == gen_routine.ErrorNormalStop {
		log.Println("terminate by", p.LogId(), "rea", rea.String(), "close", err)
	} else {
		log.Println("terminate by error", p.LogId(), "rea", rea.String(), "close", err)
	}
}

// HandleMsg 处理协程内收到的消息
func (p *Player) HandleMsg(msg1 gen_routine.Msg) (interface{}, *gen_routine.Error) {
	switch v := msg1.(type) {
	case *msg.Message:
		p.handleClientMsg(v)
		return nil, nil
	}
	return nil, nil
}

func (p *Player) handleClientMsg(v *msg.Message) {
	var ret interface{}
	switch v.Grp {
	case constant.MsgGrpChat:
		ret = p.handleChatMsg(v)
	}
	if ret != nil {
		p.Resp(v.Grp, v.Cmd, ret)
	}
}

func (p *Player) handleChatMsg(m *msg.Message) interface{} {
	switch m.Cmd {
	case constant.MsgCmdChat:
		return p.chat(m.Data.(*msg.ReqMsgChat))
	case constant.MsgCmdJoin:
		return p.join(m.Data.(*msg.ReqMsgJoin))
	}
	return nil
}

func (p *Player) Resp(grp uint8, cmd uint8, msg1 interface{}) {
	_, err := msg.Write(p.C, &msg.Message{Grp: grp, Cmd: cmd, Data: msg1})
	if err != nil {
		log.Println("send msg error", err)
	}
}

// LogId 打日志时的统一接口
func (p *Player) LogId() string {
	return fmt.Sprintf("%d", p.RoleID)
}

func (p *Player) chat(m *msg.ReqMsgChat) (ret *msg.RspMsgChat) {
	ret = &msg.RspMsgChat{Status: 0}
	if m.Content[0] == '/' {
		ret.RetStr = p.gm(m.Content)
	} else {
		str := profanity.ChangeSensitiveWords(m.Content)
		rsp := &msg.RspMsgNotify{Msg: str}
		for _, s := range gen_routine.GrpAll(p.chatGrp) {
			s.ASyncExec(s.GetMod().(*Player).Resp, uint8(constant.MsgGrpChat), uint8(constant.MsgCmdNotify), rsp)
		}
		addHistory(p.chatGrp, str)
	}
	return
}

func (p *Player) gm(cmd string) string {
	str := strings.Split(cmd, " ")
	switch str[0] {
	case "/stats":
		role, _ := strconv.Atoi(str[1])
		p1 := GetManager().GetPlayer(int64(role))
		return fmt.Sprintf("roleid %d grp %d login %d online %d", p1.RoleID, p1.chatGrp, p1.LoginStamp, time.Now().Unix()-p1.LoginStamp)
	case "/popular":
		if history.rank == nil {
			return "no word in"
		} else {
			return fmt.Sprintf("word %s times %d", history.rank.word, history.rank.times)
		}
	default:
		return "unknown"
	}
}

func (p *Player) join(m *msg.ReqMsgJoin) *msg.RspMsgJoin {
	p.GrpUnReq(p.chatGrp)
	p.chatGrp = m.Grp
	p.GrpReg(m.Grp)
	hMsg := &msg.RspMsgHistory{Msg: getHistory(p.chatGrp)}
	p.Resp(constant.MsgGrpChat, constant.MsgCmdHistory, hMsg)
	return &msg.RspMsgJoin{Status: 0}
}

func getHistory(grp int32) []string {
	history.Lock()
	defer history.Unlock()
	return history.content[grp]
}

func addHistory(grp int32, str string) {
	history.Lock()
	defer history.Unlock()
	history.content[grp] = append(history.content[grp], str)
	l := len(history.content[grp])
	if l > 50 {
		history.content[grp] = history.content[grp][l-50:]
	}
	// 词频记录
	updateFrequency(str)
}

func updateFrequency(str string) {
	// 词频记录
	str = strings.ReplaceAll(str, "*", "")
	wordL := strings.Split(str, " ")
	now := time.Now().Unix()
	for _, s := range wordL {
		old, ok := history.frequency[s]
		if !ok {
			old = &TimesElem{stamp: now}
			history.frequency[s] = old
		}
		// 清理超过当前时间10分钟的 并更新差值
		dec1 := now - old.stamp
		var times []uint16
		for _, dec := range old.times {
			if int64(dec)+dec1 > constant.MaxFSec {
				continue
			}
			times = append(times, dec+uint16(dec1))
		}
		// 记录距离时间戳多久发的内容
		old.times = append(times, 0)
		old.stamp = now
		// 更新排行
		updateRank(s, len(history.frequency[s].times))
	}
}

func updateRank(word string, times int) {
	if history.rank == nil {
		history.rank = &RankElem{word: word, times: times}
		return
	}
	if history.rank.times < times {
		history.rank.times = times
		history.rank.word = word
	}
}
