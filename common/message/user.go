package message

const (
	MAX_GROUPMGR_NUMBER    = 5
	MAX_GROUPMEMBER_NUMBER = 100
)

type RoleInGroupChat int

const (
	GroupChatOwner RoleInGroupChat = iota
	GroupChatAdmin
	GroupChatMember
)

func (this RoleInGroupChat) Visualize() string {
	if this == GroupChatOwner {
		return "群主"
	} else if this == GroupChatAdmin {
		return "群管理员"
	} else if this == GroupChatMember {
		return "群成员"
	} else {
		return "未知群身份"
	}
}

type User struct {
	UserId         int                     `json:"userId"` //规定不能为0
	UserPwd        string                  `json:"userPwd"`
	UserName       string                  `json:"userName"`
	UserStatus     int                     `json:"userStatus"`
	UserFriends    map[int]struct{}        `json:"userFriends"`
	UserGroupChats map[int]RoleInGroupChat `json:"userGroupChats"`
}

type GroupChat struct {
	GroupID     int                         `json:"groupID"`
	GroupName   string                      `json:"groupName"`
	GroupLeader int                         `json:"groupLeader"`
	GroupMgr    [MAX_GROUPMGR_NUMBER]int    `json:"groupMgr"`
	GroupMember map[int]UserInfoInGroupChat `json:"groupMember"`
}
