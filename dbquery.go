package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
)

//	"log"

func playerByID(ID uint) Player {
	var p Player
	db.First(&p, ID)
	return p
}

func userByID(ID uint) User {
	var u User
	db.First(&u, ID)
	return u
}

func eventByID(ID uint) Event {
	var e Event
	db.First(&e, ID)
	return e
}

type PGNList struct {
	PGNs []string
}

func wb1(c string) uint {
	if c == "b" {
		return 1
	}
	return 0
}

func posToFEN(pos Position) string {
	var pieces []Piece
	db.Where("position_id = ?", pos.ID).Find(&pieces)
	var chessboard [8][8]*Piece
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			chessboard[i][j] = nil
		}
	}
	for i, p := range pieces {
		column := int(p.Coord[0]) - int('a')
		row := int(p.Coord[1]) - int('1')
		chessboard[row][column] = &pieces[i]
	}
	ans := ""
	for i := 7; i >= 0; i-- {
		em := 0
		for j := 0; j < 8; j++ {
			if chessboard[i][j] == nil {
				em++
			} else {
				if em > 0 {
					ans += string(int('0') + em)
					em = 0
				}
				if chessboard[i][j].Colour == "w" {
					ans += strings.ToUpper(chessboard[i][j].Type)
				} else {
					ans += strings.ToLower(chessboard[i][j].Type)
				}
			}
		}
		if em > 0 {
			ans += string(int('0') + em)
			em = 0
		}

		if i > 0 {
			ans += "/"
		}
	}
	ans += " "
	ans += pos.SideToMove
	ans += " "
	castling := ""
	if pos.CastlingK {
		castling += "K"
	}
	if pos.CastlingQ {
		castling += "Q"
	}
	if pos.Castlingk {
		castling += "k"
	}
	if pos.Castlingq {
		castling += "q"
	}
	if len(castling) == 0 {
		castling = "-"
	}
	ans += castling
	ans += " "
	ans += pos.Enpassant
	if pos.Enpassant[1] != ' ' {
		ans += " "
	}
	ans += "0 "
	ans += strconv.Itoa(int(pos.Moveclock))
	log.Println(ans)
	return ans
}

func getGame(ID uint) PGNList {
	var positions []Position
	db.Where("game_id = ?", ID).Find(&positions)
	ans := PGNList{make([]string, len(positions))}
	for _, pos := range positions {
		i := pos.Moveclock*2 + wb1(pos.SideToMove) - 2
		ans.PGNs[i] = posToFEN(pos)
	}
	return ans
}

type GameInfo struct {
	Date   string
	Round  string
	Result string
	White  string
	Black  string
	Event  string
	Site   string
	User   string
	ID     uint
}

func getAllGames(l int, r int, queryset *gorm.DB) ([]GameInfo, int) {
	var games []Game
	queryset.Find(&games)
	if r > len(games) || r == -1 {
		r = len(games)
	}
	if l >= len(games) {
		l = len(games)
	}
	res := make([]GameInfo, r-l)
	for i := len(games) - 1 - l; i > len(games)-1-r; i-- {
		res[len(games)-1-l-i] = GameInfo{
			games[i].Date.String()[0:10],
			games[i].Round,
			games[i].Result,
			playerByID(games[i].WhiteID).Name,
			playerByID(games[i].BlackID).Name,
			eventByID(games[i].EventID).Name,
			eventByID(games[i].EventID).Site,
			userByID(games[i].UserID).Login,
			games[i].ID,
		}
	}
	return res, (len(games) + 9) / 10
}
