package main

import (
	"slices"

	"github.com/gar354/bush-campus-signin/googleLoginAuth"
	"github.com/gar354/bush-campus-signin/middleware"
	"github.com/gar354/bush-campus-signin/spreadsheetAPI"

	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var (
	// TODO: possibly move form handling, etc to another file?
	checkOutReasons = []string{
		"Free Period",
		"Family Trip",
		"Sports Dissmissal",
		"Illness",
		"Medical Appt",
		"Other Appt",
		"Lunch",
		"Other",
	}
	checkInReasons = []string{
		"Free Period",
		"Traffic",
		"Medical Appt",
		"Other Appt",
		"Bus",
		"Overslept",
		"Not Feeling Well In The Morning",
		"Lunch",
		"Other",
	}
	tpl = template.Must(template.ParseFiles(
		"templates/index.html",
		"templates/google-signin.html",
		"templates/form.html",
		"templates/post-submit.html",
	))
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	if !checkRequiredEnvVars([]string{
		"GOOGLE_OAUTH_CLIENT_ID",
		"GOOGLE_OAUTH_CLIENT_SECRET",
		"SESSION_KEY",
		"GOOGLE_SREADSHEET_ACCOUNT_EMAIL",
		"GOOGLE_SREADSHEET_ACCOUNT_KEY",
		"GOOGLE_SPREADSHEET_ID",
	}) {
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
		os.Setenv("PORT", port)
	}
	url := os.Getenv("URL")
	if url == "" {
		url = fmt.Sprintf("http://localhost:%s", port)
		os.Setenv("URL", url)
	}

	log.Printf("server hosted at %s", url)

	fs := http.FileServer(http.Dir("static"))
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/account", indexHandler)
	mux.HandleFunc("/", formHandler)
	mux.HandleFunc("/submit", formSubmitHandler)
	mux.HandleFunc("/submit/post", func(w http.ResponseWriter, h *http.Request) {
		tpl.ExecuteTemplate(w, "post-submit.html", nil)
	})

	googleLoginAuth.SetupCallbacks(mux)
	googleLoginAuth.SetupAuthConfig(
		os.Getenv("SESSION_KEY"),
		os.Getenv("URL"),
		os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
	)
	spreadsheetAPI.SetupAuthConfig(
		os.Getenv("GOOGLE_SREADSHEET_ACCOUNT_EMAIL"),
		os.Getenv("GOOGLE_SREADSHEET_ACCOUNT_KEY"),
		os.Getenv("GOOGLE_SPREADSHEET_ID"),
	)

	// HACK: unset sensitive env vars once application is setup (for security)
	os.Setenv("QR_VIEWER_PASSWORD", "")
	os.Setenv("SESSION_KEY", "")

	os.Setenv("GOOGLE_OAUTH_CLIENT_ID", "")
	os.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "")

	os.Setenv("GOOGLE_SREADSHEET_ACCOUNT_EMAIL", "")
	os.Setenv("GOOGLE_SREADSHEET_ACCOUNT_KEY", "")
	os.Setenv("GOOGLE_SPREADSHEET_ID", "")

	middlewareMux := middleware.NewProxyHandler(mux)
	if err = http.ListenAndServe(":"+port, middlewareMux); err != http.ErrServerClosed {
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
	if !googleLoginAuth.IsUserAuthenticated(r) {
		// Handle the case when the "UUID" query parameter is empty
		http.Redirect(w, r, "/account", http.StatusTemporaryRedirect)
		return
	}
	user, err := googleLoginAuth.GetUserDataFromSession(r)
	if err != nil {
		http.Error(w, "Failed to get user data", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"User":            user,
		"CheckInReasons":  checkInReasons,
		"CheckOutReasons": checkOutReasons,
	}

	tpl.ExecuteTemplate(w, "form.html", data)
}

func formSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if !googleLoginAuth.IsUserAuthenticated(r) {
		http.Error(w, "Failed to submit form: User is not authenticated", http.StatusInternalServerError)
		return
	}
	if !validateFormSubmit(r) {
		http.Error(w, "Failed to submit form: invalid form data", http.StatusInternalServerError)
		return
	}
	user, err := googleLoginAuth.GetUserDataFromSession(r)
	if err != nil {
		http.Error(w, "Failed to get user data", http.StatusInternalServerError)
		return
	}
	// submit the form data to google sheets
	err = spreadsheetAPI.SubmitSpreadSheetData(
		user.Email, r.FormValue("signin-type"),
		r.FormValue("reason"))
	if err != nil {
		http.Error(w, "Failed to record form data", http.StatusInternalServerError)
		log.Printf("Error: unabled to submit data to spreadsheet: %v", err)
		return
	}

	http.Redirect(w, r, "/submit/post", http.StatusTemporaryRedirect)
}

// TODO: if this function becomes larger, make it not stupid
func validateFormSubmit(r *http.Request) bool {
	return (r.FormValue("signin-type") == "Signing In" || r.FormValue("signin-type") == "Signing Out") &&
		(slices.Contains(checkInReasons, r.FormValue("reason")) || slices.Contains(checkOutReasons, r.FormValue("reason")))
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
