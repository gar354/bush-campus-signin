package main

import (
	"fmt"
	"gareth/attendence/googleLoginAuth"
	"gareth/attendence/spreadsheetAPI"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

var tpl = template.Must(template.ParseFiles("templates/index.html", "templates/google-signin.html", "templates/form.html"))
var currentUUID = ""
var formURL = ""

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	if googleLoginAuth.IsUserAuthenticated(r) {
		// If authenticated, get user data from session
		user, err := googleLoginAuth.GetUserDataFromSession(r)
		if err != nil {
			http.Error(w, "Failed to get user data", http.StatusInternalServerError)
			return
		}

		// Execute index.html template with user data
		err = tpl.ExecuteTemplate(w, "index.html", user)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			log.Printf("%s", err)
			return
		}
	} else {
		err := tpl.ExecuteTemplate(w, "google-signin.html", nil)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			log.Printf("%s", err)
			return
		}
	}
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	urlUUID := r.URL.Query().Get("UUID")
	if urlUUID != currentUUID || !googleLoginAuth.IsUserAuthenticated(r) {
		// Handle the case when the "UUID" query parameter is empty
		// http.Error(w, "UUID parameter is correct", http.StatusBadRequest)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	user, err := googleLoginAuth.GetUserDataFromSession(r)
	if err != nil {
		http.Error(w, "Failed to get user data", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"UUID": urlUUID,
		"User": user,
	}

	tpl.ExecuteTemplate(w, "form.html", data)
}

func formSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if !validateFormSubmit(r) {
		http.Error(w, "Failed to submit form: invalid form data", http.StatusInternalServerError)
		return
	}
	if !googleLoginAuth.IsUserAuthenticated(r) {
		http.Error(w, "Failed to submit form: User is not authenticated", http.StatusInternalServerError)
		return
	}
	user, err := googleLoginAuth.GetUserDataFromSession(r)
	if err != nil {
		http.Error(w, "Failed to get user data", http.StatusInternalServerError)
		return
	}
	spreadsheetAPI.SubmitSpreadSheetData(user.Email, r.FormValue("signin-type"), r.FormValue("free-period"), r.FormValue("reason"))
	refreshFormURL()
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func validateFormSubmit(r *http.Request) bool {
	return r.FormValue("uuid") == currentUUID && (r.FormValue("signin-type") == "Signing In" || r.FormValue("signin-type") == "Signing Out") && (r.FormValue("free-period") == "Yes" || r.FormValue("free-period") == "No") && (len(r.FormValue("reason")) <= 25)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Printf("server hosted at https://localhost:%s", port)

	fs := http.FileServer(http.Dir("static"))
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/form", formHandler)
	mux.HandleFunc("/submit", formSubmitHandler)
	googleLoginAuth.SetupCallbacks(mux)
	refreshFormURL()

	if err := http.ListenAndServeTLS(":"+port, "data/server.crt", "data/server.key", mux); err != http.ErrServerClosed {
		log.Printf("%v", err)
	} else {
		log.Println("Server closed!")
	}
}

func refreshFormURL() {
	currentUUID = uuid.NewString()
	formURL = fmt.Sprintf("https://localhost:5000/form?UUID=%s", currentUUID)
	log.Println(formURL)
	qrc, err := qrcode.New(formURL)
	if err != nil {
		log.Printf("could not generate QRCode: %v", err)
		return
	}
	w, err := standard.New("data/qrcode.jpeg")
	if err != nil {
		log.Printf("standard.New failed: %v", err)
		return
	}

	// save file
	if err = qrc.Save(w); err != nil {
		log.Printf("could not save image: %v", err)
	}
}
