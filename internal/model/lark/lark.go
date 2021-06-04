package lark

import (
	"encoding/json"
)

type DepartmentResponse struct {
	Msg  string                  `json:"msg"`
	Code int                     `json:"code"`
	Data *DepartmentResponseData `json:"data"`
}

type DepartmentResponseData struct {
	Department *DepartmentDetail `json:"department"`
}

var (
	UserChan = make(chan *ContactUserMsg, 10)
	DepChan  = make(chan *ContactDepMsg, 10)
)

type DepartmentDetail struct {
	OpenDepartmentID   string              `json:"open_department_id"`
	DepartmentID       string              `json:"department_id"`
	I18nName           *DepartmentI18nName `json:"i18n_name"`
	Name               string              `json:"name"`
	LeaderUserID       string              `json:"leader_user_id"`
	MemberCount        int                 `json:"member_count"`
	UnitIDs            []string            `json:"unit_ids"`
	ParentDepartmentID string              `json:"parent_department_id"`
	ChatID             string              `json:"chat_id"`
	Order              string              `json:"order"`
	Status             *DepartmentStatus   `json:"status"`
}

type DepartmentI18nName struct {
	EnUs string `json:"en_us"`
	ZhCn string `json:"zh_cn"`
	JaJp string `json:"ja_jp"`
}

type DepartmentStatus struct {
	IsDeleted bool `json:"is_deleted"`
}

// ContactUserMsg defines user msg
type ContactUserMsg struct {
	Schema string             `json:"schema"`
	Header *ContactMsgHeader  `json:"header"`
	Event  *ContactUserMsgEvt `json:"event"`
}

func (c *ContactUserMsg) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// ContactDepMsg defines department msg
type ContactDepMsg struct {
	Schema string            `json:"schema"`
	Header *ContactMsgHeader `json:"header"`
	Event  *ContactDepMsgEvt `json:"event"`
}

func (c *ContactDepMsg) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

type ContactMsgHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	TenantKey  string `json:"tenant_key"`
	CreateTime string `json:"create_time"`
	AppID      string `json:"app_id"`
	Token      string `json:"token"`
}

type ContactUserMsgEvt struct {
	OldObject *UserObject `json:"old_object"`
	Object    *UserObject `json:"object"`
}

type ContactDepMsgEvt struct {
	OldObject *DepObject `json:"old_object"`
	Object    *DepObject `json:"object"`
}

type DepObject struct {
	OpenDepartmentID   string `json:"open_department_id"`
	DepartmentID       string `json:"department_id"`
	Name               string `json:"name"`
	Order              int    `json:"order"`
	ParentDepartmentID string `json:"parent_department_id"`
	Status             struct {
		IsDeleted bool `json:"is_deleted"`
	} `json:"status"`
}

type UserObject struct {
	Country       string       `json:"country"`
	WorkStation   string       `json:"work_station"`
	Gender        int          `json:"gender"`
	City          string       `json:"city"`
	OpenID        string       `json:"open_id"`
	Mobile        string       `json:"mobile"`
	EmployeeNo    string       `json:"employee_no"`
	Avatar        *UserAvatar  `json:"avatar"`
	DepartmentIDs []string     `json:"department_ids"`
	JoinTime      int          `json:"join_time"`
	EmployeeType  int          `json:"employee_type"`
	UserID        string       `json:"user_id"`
	Name          string       `json:"name"`
	EnName        string       `json:"en_name"`
	Orders        []*UserOrder `json:"orders"`
	LeaderUserID  string       `json:"leader_user_id"`
	Email         string       `json:"email"`
	Status        *UserStatus  `json:"status"`
}

type UserAvatar struct {
	Avatar640    string `json:"avatar_640"`
	AvatarOrigin string `json:"avatar_origin"`
	Avatar72     string `json:"avatar_72"`
	Avatar240    string `json:"avatar_240"`
}

type UserOrder struct {
	UserOrder       int    `json:"user_order"`
	DepartmentID    string `json:"department_id"`
	DepartmentOrder int    `json:"department_order"`
}
type UserStatus struct {
	IsActivated bool `json:"is_activated"`
	IsFrozen    bool `json:"is_frozen"`
	IsResigned  bool `json:"is_resigned"`
}
