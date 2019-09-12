package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func initDB(db *gorm.DB) {
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Player{})
	db.AutoMigrate(&Event{})
	db.AutoMigrate(&Game{})
	db.AutoMigrate(&Group{})
	db.AutoMigrate(&Piece{})
	db.AutoMigrate(&Position{})
	db.Create(&User{Login: "admin", Password: "777dcf15ce360b471e23a6916c34a312", Moderator: true})
}

var db *gorm.DB

func getID(s string) uint {
	lines := strings.Split(s, "\n")

	for i, l := range lines {
		if strings.Contains(l, "name=\"ID\"") {
			v := lines[i+2][:len(lines[i+2])-1]
			u, _ := strconv.ParseUint(v, 10, 32)
			return uint(u)
		}
	}
	return 0
}

func getString(s string) string {
	lines := strings.Split(s, "\n")

	for i, l := range lines {
		if strings.Contains(l, "name=\"str\"") {
			v := lines[i+2][:len(lines[i+2])-1]
			return v
		}
	}
	return "err"

}

func Intersection(a, b []int64) (c []int64) {

	m := make(map[int64]bool)

	for _, item := range a {
		m[item] = true
	}

	for _, item := range b {
		if _, ok := m[item]; ok {
			c = append(c, item)
		}
	}
	return
}

func main() {

	var err error
	db, err = gorm.Open("postgres", "user=chessdb_user dbname=chessdb password=magnus")
	if err != nil {
		fmt.Println("err: " + err.Error())
		return
	}
	defer db.Close()
	//	initDB(db)

	http.HandleFunc("/register", register)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)

	http.HandleFunc("/filter", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.String())
		if r.URL.String() == "/filter" {
			tmpl := template.Must(template.ParseFiles("filter.html"))
			err := tmpl.Execute(w, nil)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			query := db
			fen := r.FormValue("fen")
			i := 0
			if len(fen) > 0 {

				fen = strings.Replace(fen, "/", " ", -1)
				var s [8]string
				fmt.Sscanf(fen, "%s %s %s %s %s %s %s %s", &s[0], &s[1], &s[2], &s[3], &s[4], &s[5], &s[6], &s[7])
				var posIDs []int64
				fl := true
				for c := '8'; c >= '1'; c-- {
					col := 97
					for x := 0; x < len(s[i]); x++ {
						if isDigit(s[i][x]) {
							v, _ := strconv.ParseInt(string(s[i][x]), 10, 32)
							col += int(v)
						} else {
							t := getPiece(s[i][x]).Type
							coord := string(col) + string(c)
							var pieces []Piece
							var ids []int64
							db.Where("coord = ? AND type = ?", coord, t).Find(&pieces).Pluck("position_id", &ids)
							if fl {
								fl = false
								posIDs = ids
							} else {
								posIDs = Intersection(posIDs, ids)
							}
							col++
						}
					}
					i++
				}
				log.Println(posIDs)
				var gamesIDs []int64
				db.Model(&Position{}).Where(posIDs).Pluck("game_id", &gamesIDs)
				log.Println(gamesIDs)
				query = query.Where(gamesIDs)
			}
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
					http.Error(w, "Bad query", 404)
					return
				}
			} else if len(event) > 0 {
				http.Error(w, "Bad query", 404)
				return
			}
			date1 := r.FormValue("dateF")
			date2 := r.FormValue("dateT")
			round := r.FormValue("round")

			if len(white) > 0 {
				var P Player
				db.Where("name = ?", white).First(&P)
				if P.ID == 0 {
					http.Error(w, "Bad query", 404)
					return
				} else {
					query = query.Where("white_id = ?", P.ID)
				}
			}
			if len(black) > 0 {
				var P Player
				db.Where("name = ?", black).First(&P)
				if P.ID == 0 {
					http.Error(w, "Bad query", 404)
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
					http.Error(w, "Bad query = query", 404)
					return
				} else {
					query = query.Where("event_id = ?", E.ID)
				}
				if len(round) > 0 {
					query = query.Where("round = ?", round)
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
			}{auth, username, gms, sz})
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	})

	http.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			id, err := strconv.ParseInt(r.FormValue("delete"), 10, 32)
			if err != nil {
				log.Println(err)
			}
			log.Println("delete game id = ", id)

			db.Where("id = ?", id).Delete(Game{})
			db.Where("game_id = ?", id).Delete(Position{})
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})
	http.HandleFunc("/getPlayer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)

			s := buf.String()
			s = getString(s)

			log.Println("getPlayer " + s)

			var players []Player
			db.Where("lower(name) LIKE ?", strings.ToLower(s)+"%").Find(&players)
			log.Println(players)
			for _, pl := range players {
				w.Write([]byte(pl.Name + "\n"))
			}
		}
	})

	http.HandleFunc("/getEvent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)

			s := buf.String()
			s = getString(s)

			log.Println("getEvent " + s)

			var events []Event
			db.Where("lower(name) LIKE ?", strings.ToLower(s)+"%").Find(&events)
			log.Println(events)
			for _, ev := range events {
				w.Write([]byte(ev.Name + " " + strconv.Itoa(ev.Year) + "\n"))
			}
		}
	})

	http.HandleFunc("/getGame", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)

			s := buf.String()

			ID := getID(s)
			log.Println("game id = ", ID)

			game := getGame(ID)

			for _, fen := range game.PGNs {
				w.Write([]byte(fen + "X"))
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "chessdb")
		if r.Method == http.MethodPost {

			if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
				http.Error(w, "Only registered users can add games", http.StatusForbidden)
				return
			}

			log.Println("new game", r.FormValue("white"), "vs", r.FormValue("black"))

			var user User
			db.Where("login = ?", session.Values["username"].(string)).First(&user)

			dateString := r.FormValue("date")
			date, err := time.Parse("2006-01-02", dateString)

			if err != nil {
				log.Println(err.Error())
			}

			err = addGame(pgnToGame(r.FormValue("pgn")),
				r.FormValue("white"),
				r.FormValue("black"),
				r.FormValue("event"),
				r.FormValue("site"),
				r.FormValue("round"),
				r.FormValue("result"),
				date,
				user, nil)
			if err != nil {
				log.Panicln(err.Error())
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return

		}

		tmpl := template.Must(template.ParseFiles("layout.html"))
		url := r.URL.String()
		if url == "/favicon.ico" {
			http.Redirect(w, r, "/img/favicon.jpg", http.StatusSeeOther)
			return
		}
		log.Println(url)

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

		gms, sz := getAllGames(int(p*10), int(p*10+10), db)

		err = tmpl.Execute(w, struct {
			LoggedIn bool
			Username string
			Games    []GameInfo
			Pages    int
		}{auth, username, gms, sz})
		if err != nil {
			fmt.Println(err.Error())
		}
	})

	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))

	fmt.Println(":9000")
	err = http.ListenAndServe(":9000", nil)
	if err != nil {
		fmt.Println(err)
	}

}

/*            {{range .Games}}
if ({{.Id}}===id) {
    infoEl.html("{{.White}} vs. {{.Black}}\n added by {{.AddedUser}}")
    game.load_pgn({{.Pgn}})

    toBegin()
    toEnd()


    ext_pgn = game.pgn()
    updateBoard(board,false)
}
{{end}}*/
