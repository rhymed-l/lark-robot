package model

import "time"

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OpenID         string    `gorm:"size:100;uniqueIndex;not null" json:"open_id"`
	UnionID        string    `gorm:"size:100" json:"union_id"`
	UserID         string    `gorm:"size:100" json:"user_id"`
	Name           string    `gorm:"size:255" json:"name"`
	EnName         string    `gorm:"size:255" json:"en_name"`
	Avatar         string    `gorm:"size:500" json:"avatar"`
	Description    string    `gorm:"type:text" json:"description"`
	Email          string    `gorm:"size:255" json:"email"`
	City           string    `gorm:"size:255" json:"city"`
	JobTitle       string    `gorm:"size:255" json:"job_title"`
	WorkStation    string    `gorm:"size:255" json:"work_station"`
	EmployeeNo     string    `gorm:"size:50" json:"employee_no"`
	Gender         int       `json:"gender"`
	LeaderUserID   string    `gorm:"size:100" json:"leader_user_id"`
	DepartmentIDs   string    `gorm:"type:text" json:"department_ids"`
	DepartmentNames string    `gorm:"type:text" json:"department_names"`
	CustomAttrs     string    `gorm:"type:text" json:"custom_attrs"`
	JoinTime       int64     `json:"join_time"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	MsgCount       int64     `json:"msg_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
