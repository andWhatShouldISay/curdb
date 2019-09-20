package main

import (
	"fmt"
	"log"
	"os"

	//	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/freeeve/pgn"
	"github.com/jinzhu/gorm"
)

func isDigit(c uint8) bool {
	return '0' <= c && c <= '9'
}

func getPiece(c uint8) Piece {
	if 'a' <= c && c <= 'z' {
		return Piece{Colour: "b", Type: string(c)}
	}
	return Piece{Colour: "w", Type: strings.ToLower(string(c))}
}

func parseFEN(fen string, gameID uint, tx *gorm.DB) {
	fen = strings.Replace(fen, "/", " ", -1)
	var s [8]string
	pos := Position{GameID: gameID}
	var castling string
	var X int

	fmt.Sscanf(fen, "%s %s %s %s %s %s %s %s %s %s %s %d %d", &s[0], &s[1], &s[2], &s[3], &s[4], &s[5], &s[6], &s[7],
		&pos.SideToMove, &castling, &pos.Enpassant, &X, &pos.Moveclock)

	pos.CastlingK = strings.Contains(castling, "K")
	pos.CastlingQ = strings.Contains(castling, "Q")
	pos.Castlingk = strings.Contains(castling, "k")
	pos.Castlingq = strings.Contains(castling, "q")
	tx.Create(&pos)
	i := 0
	for c := '8'; c >= '1'; c-- {
		col := 97
		for x := 0; x < len(s[i]); x++ {
			if isDigit(s[i][x]) {
				v, _ := strconv.ParseInt(string(s[i][x]), 10, 32)
				col += int(v)
			} else {
				piece := getPiece(s[i][x])
				piece.Coord = string(col) + string(c)
				piece.PositionID = pos.ID
				tx.Create(&piece)
				col++
			}
		}

		i++
	}
}

func addGamesFromFile(filename string) int {
	log.Println(filename)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	ps := pgn.NewPGNScanner(f)
	k := 0

	// while there's more to read in the file
	for ps.Next() {
		// scan the next game
		game, err := ps.Scan()

		ratingWhite, _ := strconv.ParseInt(game.Tags["WhiteElo"], 10, 32)
		ratingBlack, _ := strconv.ParseInt(game.Tags["BlackElo"], 10, 32)
		ratingBound := int64(2600)

		if err != nil && ratingBlack >= ratingBound && ratingWhite >= ratingBound {
			log.Fatal(game.Tags["White"], game.Tags["Black"], err)
		}
		if ratingBlack >= ratingBound && ratingWhite >= ratingBound {
			k++
			t, err := time.Parse("2006.1.2", strings.Replace(game.Tags["Date"], "??", "01", -1))
			if err != nil {
				log.Println(game.Tags["Date"], err.Error())
			}

			tx := db.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()
			addGame(*game, game.Tags["White"], game.Tags["Black"], game.Tags["Event"], game.Tags["Site"],
				game.Tags["Round"], game.Tags["Result"], t, User{}, tx)
			tx.Commit()

		}
	}
	return k
}

func pgnToGame(pgn_ string) pgn.Game {
	ps := pgn.NewPGNScanner(strings.NewReader(pgn_))
	game, err := ps.Scan()
	if err != nil {
		log.Println(err.Error())
	}
	return *game
}

func addGame(game pgn.Game, white string, black string, eventname string, site string, round string, result string, date time.Time, user User, tx *gorm.DB) error {
	if result == "1/2" {
		result = "1/2-1/2"
	}
	var whitePlayer, blackPlayer Player
	flag := false
	if tx == nil {
		tx = db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		flag = true
	}

	if err := tx.Error; err != nil {
		return err
	}

	db.Where("name = ?", white).First(&whitePlayer)
	if whitePlayer.ID == 0 {
		whitePlayer.Name = white
		if err := tx.Create(&whitePlayer).Error; err != nil {
			tx.Rollback()
			return err
		}
		log.Println("created player ", whitePlayer.Name, whitePlayer.ID)
	}

	db.Where("name = ?", black).First(&blackPlayer)
	if blackPlayer.ID == 0 {
		blackPlayer.Name = black
		if err := tx.Create(&blackPlayer).Error; err != nil {
			tx.Rollback()
			return err
		}
		log.Println("created player ", blackPlayer.Name, blackPlayer.ID)
	}

	var event Event
	if len(eventname) > 0 {
		db.Where("name = ? and year = ?", eventname, date.Year()).First(&event)
		if event.ID == 0 {
			event.Name = eventname
			event.Year = date.Year()
			event.Site = site
			if err := tx.Create(&event).Error; err != nil {
				tx.Rollback()
				return err
			}
			log.Println("created event ", event.Name, event.Year)
		} else if site != event.Site {
			log.Println("event " + event.Name + " was hosted in " + event.Site + ", not in " + site)
		}
	}

	g := Game{Date: date, Round: round, Result: result, UserID: 1,
		WhiteID: whitePlayer.ID, BlackID: blackPlayer.ID, EventID: event.ID}

	if err := tx.Create(&g).Error; err != nil {
		tx.Rollback()
		return err
	}
	cash = make(map[string]([]int64))

	b := pgn.NewBoard()
	parseFEN(b.String(), g.ID, tx)
	for _, move := range game.Moves {
		b.MakeMove(move)
		parseFEN(b.String(), g.ID, tx)
	}

	if !flag {
		return nil
	} else {
		return tx.Commit().Error
	}
}
