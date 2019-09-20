package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
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
	return ans
}

func getGame(ID uint) PGNList {
	var positions []Position
	db.Where("game_id = ?", ID).Find(&positions)
	ans := PGNList{make([]string, len(positions))}
	for _, pos := range positions {
		i := pos.Moveclock*2 + wb1(pos.SideToMove) - 2
		ans.PGNs[i] = posToFEN(pos)
		log.Println(i, ans.PGNs[i])
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
	queryset.Order("id").Find(&games)
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

var cash map[string]([]int64)

func filter(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.String())
	if r.URL.String() == "/filter" {
		tmpl := template.Must(template.ParseFiles("filter.html"))
		err := tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		query := db
		fen := r.FormValue("fen")
		i := 0
		if fen != "" {
			fen = strings.Replace(fen, "/", " ", -1)
			var gamesIDs []int64
			_, prs := cash[fen]
			if prs {
				gamesIDs = cash[fen]
			} else {
				var s [8]string
				fmt.Sscanf(fen, "%s %s %s %s %s %s %s %s", &s[0], &s[1], &s[2], &s[3], &s[4], &s[5], &s[6], &s[7])
				var pieces []Piece
				for c := '8'; c >= '1'; c-- {
					col := 97
					for x := 0; x < len(s[i]); x++ {
						if isDigit(s[i][x]) {
							v, _ := strconv.ParseInt(string(s[i][x]), 10, 32)
							col += int(v)
						} else {
							t := getPiece(s[i][x]).Type
							clr := getPiece(s[i][x]).Colour
							coord := string(col) + string(c)
							pieces = append(pieces, Piece{Type: t, Colour: clr, Coord: coord})
							col++
						}
					}
					i++
				}

				if len(pieces) > 0 {

					query = query.Table("pieces").Select("position_id, COUNT(position_id) as B")
					var posIDs []int64

					for _, p := range pieces {
						query = query.Or("coord = ? AND type = ? AND colour = ?", p.Coord, p.Type, p.Colour)
					}
					query = query.Group("position_id").Having("COUNT(position_id) = ? ", len(pieces))
					log.Println(query.SubQuery())

					rows, err := query.Rows()

					if err != nil {
						log.Println(err.Error())
					}

					kol := 0
					for rows.Next() {
						var posID int64
						var X int64
						rows.Scan(&posID, &X)
						posIDs = append(posIDs, posID)
						kol++
					}
					log.Println(kol)
					query = db

					if len(pieces) < 32 {
						for i := 0; i < len(posIDs); i++ {
							var count int
							query.Model(&Piece{}).Where("position_id = ?", posIDs[i]).Count(&count)
							if count != len(pieces) {
								posIDs[i] = 0
							}
						}
					}
					db.Model(&Position{}).Where(posIDs).Pluck("game_id", &gamesIDs)
					if len(cash) == 100 {
						cash = make(map[string]([]int64))
					}
					cash[fen] = gamesIDs
				}
			}
			query = query.Where(gamesIDs)
		} else {
			result := r.FormValue("result")
			if result == "1/2" {
				result = "1/2-1/2"
			}
			white := r.FormValue("white-choice")
			black := r.FormValue("black-choice")
			event := r.FormValue("event-choice")
			year := int64(0)
			if len(event) > 5 {
				var err error
				year, err = strconv.ParseInt(event[len(event)-4:], 10, 32)
				event = event[:len(event)-5]
				if err != nil {
					http.Error(w, "Bad event", 404)
					return
				}
			} else if len(event) > 0 {
				http.Error(w, "Bad event", 404)
				return
			}
			date1 := r.FormValue("dateF")
			date2 := r.FormValue("dateT")
			round := r.FormValue("round")

			if len(white) > 0 {
				var P Player
				db.Where("name = ?", white).First(&P)
				if P.ID == 0 {
					http.Error(w, "Bad white player", 404)
					return
				} else {
					query = query.Where("white_id = ?", P.ID)
				}
			}
			if len(black) > 0 {
				var P Player
				db.Where("name = ?", black).First(&P)
				if P.ID == 0 {
					http.Error(w, "Bad black player", 404)
					return
				} else {
					query = query.Where("black_id = ?", P.ID)
				}
			}
			if len(result) > 0 {
				query = query.Where("result = ?", result)
			}
			if len(date1) > 0 {
				query = query.Where("date >= ?", date1)
			}
			if len(date2) > 0 {
				query = query.Where("date <= ?", date2)
			}
			if len(event) > 0 && year > 0 {
				var E Event
				db.Where("name = ? and year = ?", event, year).First(&E)
				if E.ID == 0 {
					http.Error(w, "Bad event", 404)
					return
				} else {
					query = query.Where("event_id = ?", E.ID)
				}
				if len(round) > 0 {
					query = query.Where("round = ?", round)
				}
			}
		}
		session, _ := store.Get(r, "chessdb")
		tmpl := template.Must(template.ParseFiles("layout.html"))

		auth, ok := session.Values["authenticated"].(bool)
		username := ""

		if auth && ok {
			username, _ = session.Values["username"].(string)
		}
		pg := r.FormValue("page")
		if pg == "" {
			pg = "0"
		}
		p, err := strconv.ParseInt(pg, 10, 32)
		if err != nil {
			p = 0
		}

		gms, sz := getAllGames(int(p*10), int(p*10+10), query)

		err = tmpl.Execute(w, struct {
			LoggedIn bool
			Username string
			Games    []GameInfo
			Pages    int
			Allow    bool
		}{auth, username, gms, sz, username == "admin"})
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}
