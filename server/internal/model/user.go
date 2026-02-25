package model

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:32;not null" json:"username"`
	Password  string    `gorm:"size:128;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type CardDescription struct {
	Type        string `gorm:"primaryKey;size:32" json:"type"`
	Name        string `gorm:"size:32;not null" json:"name"`
	Description string `gorm:"size:255;not null" json:"description"`
}

func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &CardDescription{}); err != nil {
		return nil, err
	}
	
	initCardDescriptions(db)

	return db, nil
}

func initCardDescriptions(db *gorm.DB) {
	descriptions := []CardDescription{
		{Type: "accuse_1", Name: "指控(1)", Description: "对目标玩家增加1点指控值"},
		{Type: "accuse_2", Name: "指控(2)", Description: "对目标玩家增加2点指控值"},
		{Type: "accuse_3", Name: "指控(3)", Description: "对目标玩家增加3点指控值"},
		{Type: "night", Name: "夜晚", Description: "强制进入夜晚阶段"},
		{Type: "contagion", Name: "传染", Description: "所有玩家从左边玩家获取一张身份牌"},
		{Type: "black_cat", Name: "黑猫", Description: "持有者在传染时被迫翻开一张身份牌"},
		{Type: "sanctuary", Name: "避难所", Description: "保护持有者免受女巫杀害"},
		{Type: "devotee", Name: "信徒", Description: "保护持有者免受指控卡影响"},
		{Type: "frame", Name: "嫁祸", Description: "将目标玩家的指控转移给另一个玩家"},
		{Type: "arson", Name: "纵火", Description: "弃置目标玩家的所有手牌"},
		{Type: "detention", Name: "拘留", Description: "跳过目标玩家的下一个回合"},
		{Type: "defense", Name: "辩护", Description: "移除目标玩家的指控值"},
		{Type: "robbery", Name: "抢劫", Description: "将目标玩家的所有手牌转移给另一个玩家"},
		{Type: "curse", Name: "诅咒", Description: "移除目标玩家的一张装备牌"},
	}

	for _, d := range descriptions {
		var count int64
		db.Model(&CardDescription{}).Where("type = ?", d.Type).Count(&count)
		if count == 0 {
			db.Create(&d)
		}
	}
}
