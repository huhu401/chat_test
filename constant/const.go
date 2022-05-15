package constant

const (
	Vsn = 1 //版本当前默认使用1
)

const (
	NetHeaderLen = 2
	GrpBase      = 1000
)

// 消息定义 组
const (
	MsgGrpLogin = 1
	MsgGrpChat  = 2
)

// 消息定义 登录子命令
const (
	MsgCmdLogin = 1
)

// 消息定义 聊天子命令
const (
	MsgCmdChat    = 1
	MsgCmdJoin    = 2
	MsgCmdNotify  = 3
	MsgCmdHistory = 4
)

// 错误码
const (
	ErrorNo         = 0
	ErrorFirstLogin = -100001 // 还未登录
)
