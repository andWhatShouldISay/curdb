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

	"github.com/gorilla/csrf"
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

func main() {

	var err error
	db, err = gorm.Open("postgres", "user=chessdb_user dbname=chessdb password=magnus")
	if err != nil {
		fmt.Println("err: " + err.Error())
		return
	}
	defer db.Close()
	//
	initDB(db)

	cash = make(map[string]([]int64))

	http.HandleFunc("/register", register)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)

	http.HandleFunc("/filter", filter)

	http.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			id, err := strconv.ParseInt(r.FormValue("delete"), 10, 32)
			if err != nil {
				log.Println(err)
			}
			log.Println("delete game id = ", id)

			db.Where("id = ?", id).Delete(Game{})
			var posIDs []int64
			db.Where("game_id = ?", id).Find(&posIDs)
			db.Where("game_id = ?", id).Delete(Position{})
			db.Where("position_id in (?)", posIDs).Delete(Piece{})
			cash = make(map[string]([]int64))
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
			LoggedIn  bool
			Username  string
			Games     []GameInfo
			Pages     int
			Allow     bool
			CsrfField template.HTML
		}{auth, username, gms, sz, userByName(username).Moderator, csrf.TemplateField(r)})
		if err != nil {
			fmt.Println(err.Error())
		}
	})

	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))

	fmt.Println(":7000")
	err = http.ListenAndServe(":7000", csrf.Protect([]byte("0123456789ABCDEF"), csrf.Secure(false))(http.DefaultServeMux))
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
