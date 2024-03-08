package main

import (
	"gareth/attendence/googleLoginAuth"
	"gareth/attendence/serveQr"
	"gareth/attendence/spreadsheetAPI"

	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var tpl = template.Must(template.ParseFiles(
	"templates/index.html",
	"templates/google-signin.html",
	"templates/form.html",
	"templates/qr-viewer.html"))

var qrServer serveQr.Server

var wsPassword string = uuid.NewString()
func main() {
	/* if you pass in generate-sheets-token, it will generate the token required
	to access the google sheets API for said user */
	if !checkRequiredEnvVars([]string{
		"GOOGLE_SPREADSHEET_OAUTH_CLIENT_ID",
		"GOOGLE_SPREADSHEET_OAUTH_CLIENT_SECRET",
	}) {
		return
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "generate-sheets-token" {
			spreadsheetAPI.SaveTokenFromWeb()
			return
		}
	}

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	if !checkRequiredEnvVars([]string{
		"GOOGLE_OAUTH_CLIENT_ID",
		"GOOGLE_OAUTH_CLIENT_SECRET",
		"GOOGLE_SPREADSHEET_ID",
		"QR_VIEWER_PASSWORD",
	}) {
		return
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
	mux.HandleFunc("/qr-viewer", qrViewHandler)
	mux.HandleFunc("/qr", qrWSHandler)
	googleLoginAuth.SetupCallbacks(mux)

	qrServer = serveQr.New()
	go qrServer.Broadcast.Serve()
	if err = http.ListenAndServeTLS(":"+port, "data/server.crt", "data/server.key", mux); err != http.ErrServerClosed {
		log.Printf("%v", err)
	} else {
		log.Println("Server closed!")
	}
}

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
	if urlUUID != qrServer.GetUUID() || !googleLoginAuth.IsUserAuthenticated(r) {
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
	err = spreadsheetAPI.SubmitSpreadSheetData(
		user.Email, r.FormValue("signin-type"),
		r.FormValue("free-period"),
		r.FormValue("reason"))
	if err != nil {
		log.Printf("Unable to submit data to spreadsheet: %v", err)
	}

	go qrServer.RefreshQr()
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func validateFormSubmit(r *http.Request) bool {
	return r.FormValue("uuid") == qrServer.GetUUID() &&
		(r.FormValue("signin-type") == "Signing In" || r.FormValue("signin-type") == "Signing Out") &&
		(r.FormValue("free-period") == "Yes" || r.FormValue("free-period") == "No") &&
		(len(r.FormValue("reason")) <= 25)
}

func qrWSHandler(w http.ResponseWriter, r *http.Request) {
	password := r.URL.Query().Get(wsPassword)
	if password != wsPassword {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println("Unauthorized http request for QR image rejected.")
		return
	}

	conn, err := qrServer.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Send QR code image data on first connect
	if err := conn.WriteMessage(websocket.BinaryMessage, qrServer.GetIMGData()); err != nil {
		log.Println("Failed to send QR code data:", err)
		return
	}

	client := qrServer.Broadcast.Register()

	go func() {
		for {
			select {
			case newData, ok := <-client:
				if !ok {
					log.Println("client channel closed!")
					return
				}
				log.Println("successfully broadcasted new data")
				if err := conn.WriteMessage(websocket.BinaryMessage, newData); err != nil {
					log.Println("Failed to send QR code data:", err)
				}
			}
		}
	}()
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			qrServer.Broadcast.DeRegister(client)
			log.Println("WebSocket connection closed by client:", err)
			break
		}
	}
}

func qrViewHandler(w http.ResponseWriter, r *http.Request) {
	password := r.URL.Query().Get("password")
	if password != os.Getenv("QR_VIEWER_PASSWORD") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println("Unauthorized http request for QR image viewer rejected.")
		return
	}
	tpl.ExecuteTemplate(w, "qr-viewer.html", wsPassword)
}

func checkRequiredEnvVars(requiredEnvVars []string) bool {
	for _, envVar := range requiredEnvVars {
		value := os.Getenv(envVar)
		if value == "" {
			log.Fatalf("Could not find required env var: %s", envVar)
			return false
		}
	}
	return true
}
