package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type Project struct {
	ProjectID        uint64 `gorm:"column:id"`
	UserName         string `json:"username" gorm:"column:project_user_name"`
	University       string `json:"university" gorm:"column:project_university"`
	College          string `json:"college" gorm:"column:project_college"`
	Major            string `json:"major" gorm:"column:project_major"`
	Email            string `json:"email" gorm:"column:project_email"`
	Phone            string `json:"phone" gorm:"column:project_phone"`
	ProjectDirection string `json:"projectDirection" gorm:"column:projectDirection"`
}
type ProjectDetail struct {
	ProjectDetailID     uint64  `gorm:"column:id"`
	ProjectDetailSort   string  `json:"projectSort" gorm:"column:project_detail_sort"`
	ProjectDetailName   string  `json:"projectName" gorm:"column:project_detail_name"`
	ProjectDetailPerson Person  `json:"Member" gorm:"column:project_detail_person;type:text"`
	ProjectDetailIntro  string  `json:"Introduction" gorm:"column:project_detail_intro"`
	ProjectDetailIdea   string  `json:"Creativity" gorm:"column:project_detail_idea"`
	ProjectDetailAdv    string  `json:"Advantage" gorm:"column:project_detail_adv"`
	ProjectDetailTea    TPerson `json:"Instructor" gorm:"column:project_detail_teacher;type:text"`
}
type Person []string
type TPerson []string

func (p *Person) Scan(value interface{}) error {
	byteValue, _ := value.([]byte)
	return json.Unmarshal(byteValue, p)
}
func (p Person) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// 实现 Scanner 接口方法
func (t *TPerson) Scan(value interface{}) error {
	byteValue, ok := value.([]byte)
	if !ok {
		return errors.New("invalid TPerson value")
	}

	return json.Unmarshal(byteValue, t)
}

// 实现 Valuer 接口方法
func (t TPerson) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (p *Project) UnmarshalJSON(data []byte) error {
	required := struct {
		UserName         string `json:"username"`
		University       string `json:"university"`
		College          string `json:"college"`
		Major            string `json:"major"`
		Email            string `json:"email"`
		Phone            string `json:"phone"`
		ProjectDirection string `json:"projectDirection"`
		Status           string `json:"status"`
	}{}
	err := json.Unmarshal(data, &required)
	if err != nil {
		return fmt.Errorf("将 JSON 数据解码进入结构体失败: %s", err.Error())
	}
	p.UserName = required.UserName
	p.University = required.University
	p.College = required.College
	p.Major = required.Major
	p.Email = required.Email
	p.Phone = required.Phone
	p.ProjectDirection = required.ProjectDirection
	return nil
}
func (p Project) TableName() string {
	return "project"
}
func (p ProjectDetail) TableName() string {
	return "project1"
}
