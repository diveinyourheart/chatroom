package message

type Friend struct {
	FriendId       int
	FriendName     string
	FriendStatus   int
	FriendNoteName string
}

type UserInfoInGroupChat struct {
	NickNameInGC string          `json:"nickNameInGC"`
	Role         RoleInGroupChat `json:"role"`
}
