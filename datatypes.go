package main

import (
	"time"

	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model
	Login     string `gorm:"type:text;not null;unique"`
	Password  string `gorm:"type:char(32);not null"`
	Moderator bool   `gorm:"not null"`
	Games     []Game
	Groups    []Group
}

type Player struct {
	gorm.Model
	Name         string `gorm:"type:text;not null"`
	GamesAsWhite []Game `gorm:"foreignkey:WhiteID"`
	GamesAsBlack []Game `gorm:"foreignkey:BlackID"`
}

type Event struct {
	gorm.Model
	Name  string `gorm:"type:text"`
	Year  int
	Site  string `gorm:"type:text"`
	Games []Game
}

type Group struct {
	gorm.Model
	UserID uint   `gorm:"not null"`
	Games  []Game `gorm:"many2many:group_games;"`
}

type Game struct {
	gorm.Model
	Date      time.Time `gorm:"type:date"`
	Round     string    `gorm:"type:text"`
	Result    string    `gorm:"type:char(7);default:'*'"`
	UserID    uint      `gorm:"not null"`
	WhiteID   uint
	BlackID   uint
	EventID   uint
	Positions []Position `gorm:"foreignkey:GameID"`
}

type Position struct {
	gorm.Model
	Moveclock  uint    `gorm:"not null"`
	SideToMove string  `gorm:"type:char(1);not null"`
	CastlingK  bool    `gorm:"not null"`
	CastlingQ  bool    `gorm:"not null"`
	Castlingk  bool    `gorm:"not null"`
	Castlingq  bool    `gorm:"not null"`
	Enpassant  string  `gorm:"type:char(2)"`
	GameID     uint    `gorm:"not null"`
	Pieces     []Piece `gorm:"foreignkey:PositionID"`
}

type Piece struct {
	gorm.Model
	PositionID uint   `gorm:"not null"`
	Type       string `gorm:"type:char(1);not null"`
	Colour     string `gorm:"type:char(1);not null"`
	Coord      string `gorm:"type:char(2);not null"`
}
