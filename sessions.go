// sessions.go
package main

import (
	"crypto/md5"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("surbr-se80et-key")
	store = sessions.NewCookieStore(key)
)

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		session, _ := store.Get(r, "chessdb")

		userLogin := r.FormValue("login")


		passwordBytes := md5.Sum([]byte(r.FormValue("password")))
		password := hex.EncodeToString(passwordBytes[:])

		log.Println("login", userLogin, password)

		var user User

		db.Where("login = ?", userLogin).Find(&user)


		if user.ID==0 {
			log.Println("no user")
			template.Must(template.ParseFiles("login.html")).Execute(w, struct{ Success bool }{false})
			return
		}

		if user.Password != password {
			log.Println("wrong password")
			template.Must(template.ParseFiles("login.html")).Execute(w, struct{ Success bool }{false})
			return
		}
		log.Println("successful login")

		// Set user as authenticated
		session.Values["authenticated"] = true
		session.Values["username"] = userLogin
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusSeeOther)

	} else {
		template.Must(template.ParseFiles("login.html")).Execute(w, struct{ Success bool }{true})
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "chessdb")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Values["username"] = ""
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		userLogin := r.FormValue("login")
		passwordBytes := md5.Sum([]byte(r.FormValue("password")))
		password := hex.EncodeToString(passwordBytes[:])


		log.Println("register", userLogin, password)

		var user User

		db.Where("login = ?", userLogin).Find(&user)

		if user.ID!=0 {
			log.Println("user already exist")
			template.Must(template.ParseFiles("register.html")).Execute(w, struct{ Success bool }{false})
			return
		} else {
			log.Println("succesfully registered")
			db.Create(&User{Login:userLogin,Password:password,Moderator:false})
		}




		login(w, r)
	} else {
		template.Must(template.ParseFiles("register.html")).Execute(w, struct{ Success bool }{true})
	}
}
