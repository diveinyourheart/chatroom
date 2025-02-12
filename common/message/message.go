package message

import (
	"time"
)

// Message中的消息类型常量
const (
	LoginMesType                        = "LoginMes"
	LoginResMesType                     = "LoginResMes"
	RegisterMesType                     = "RegisterMes"
	RegisterResMesType                  = "RegisterResMes"
	NotifyUserStatusMesType             = "NotifyUserStatusMes"
	ServerReadClientPkgFailureType      = "ServerReadClientPkgFailure"
	OneToOneMesType                     = "OneToOneMes"
	AddFriendMesType                    = "AddFriendMes"
	AddFriendResMesType                 = "AddFriendResMes"
	SendingOneToOneMesFailureNoticeType = "SendingOneToOneMesFailureNotice"
	LoggedInOnAnotherDeviceType         = "LoggedInOnAnotherDevice"
	GroupManageMesType                  = "GroupManageMes"
	GroupManageResMesType               = "GroupManageResMes"
	GroupChatMesType                    = "GroupChatMes"
)

// 对话消息当前是发送阶段还是接收阶段
const (
	TX__ string = "transmission"
	RX__ string = "reception"
)

// 用户在线状态
const (
	UserOffline = iota
	UserOnline
	UserBusyStatus
)

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type LoginMes struct {
	UserId   int    `json:"userId"`
	UserPwd  string `json:"userPwd"`
	UserName string `json:"userName"`
}

type LoginResMes struct {
	Code       int         `json:"code"`  //返回状态码，500表示该用户还未注册，200表示登录成功
	Error      string      `json:"error"` //返回错误信息
	Usr        User        `json:"usr"`
	Friends    []Friend    `json:"friends"`
	GroupChats []GroupChat `json:"groupChats"`
}

type RegisterMes struct {
	Usr User `json:"user"`
}

type RegisterResMes struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// 为了配合服务器端推送用户状态变化的消息
type NotifyUserStatusMes struct {
	UserId int `json:"userId"`
	Status int `json:"status"`
}

// 私聊消息结构体定义
type OneToOneMes struct {
	Content     string    `json:"content"`
	DesId       int       `json:"userId"`
	OriginId    int       `json:"originId"`
	SendingTime time.Time `json:"sendingTime"`
	TXorRX      string
}

type AddFriendMes struct {
	Requester    Friend `json:"requester"`
	TargetUserID int    `json:"targetUserID"`
	Note         string `json:"note"`
}

type AddFriendResMes struct {
	FriendInfo   Friend `json:"friendInfo"`
	TargetUserID int    `json:"targetUserID"`
	IsAgree      bool   `json:"isAgree"`
}

type SendingOneToOneMesFailureNotice struct {
	DesId int    `json:"desId"`
	Error string `json:"error"`
}

type LoggedInOnAnotherDevice struct {
	LoginTime       time.Time
	OperatingSystem string
	HostName        string
}

type manageMesType int

const (
	CREATE_A_GROUP_CHAT manageMesType = iota
	ADD_ADMINISTRATOR
	JOIN_GROUP_CHAT
)

type GroupManageMes struct {
	ManageMesType   manageMesType `json:"manageMesType"`
	OperandID       int           `json:"operatorID"`
	GroupChatID     int           `json:"groupChatID"`
	ManageInfo      string        `json:"manageInfo"`
	OperandInfo     Friend        `json:"operandInfo"`
	JoinRequestTime time.Time     `json:"joinRequestTime"` //仅ManageMesType = JOIN_GROUP_CHAT会用到此字段
}

type GroupManageResMes struct {
	Code            int                 `json:"code"`
	GroupChatInfo   GroupChat           `json:"groupChatInfo"`
	Error           string              `json:"error"`
	ManageMesType   manageMesType       `json:"manageMesType"`
	OperandID       int                 `json:"operandID"`
	GroupChatID     int                 `json:"groupChatID"`
	IsApproved      bool                `json:"isApproved"`      //仅ManageMesType = JOIN_GROUP_CHAT会用到此字段
	DecidedBy       int                 `json:"decidedBy"`       //仅ManageMesType = JOIN_GROUP_CHAT会用到此字段
	JoinRequestTime time.Time           `json:"joinRequestTime"` //仅ManageMesType = JOIN_GROUP_CHAT会用到此字段
	NewUserInfoInGC UserInfoInGroupChat `json:"NewUserInfoInGC"` //仅ManageMesType = JOIN_GROUP_CHAT会用到此字段
}

type GroupChatMes struct {
	OriginId    int       `json:"originId"`
	GroupChatId int       `json:"groupChatId"`
	SendingTime time.Time `json:"sendingTime"`
	Content     string    `json:"content"`
	TXorRX      string
}
