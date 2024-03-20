package googleLoginAuth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type User struct {
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Picture       string `json:"picture"`
}

var store *sessions.FilesystemStore

// Scopes: OAuth 2.0 scopes provide a way to limit the amount of access that is granted to an access token.
var googleOauthConfig *oauth2.Config

func SetupAuthConfig(sessionKey string, url string, clientID string, clientSecret string) {
	store = sessions.NewFilesystemStore("data/", []byte(sessionKey))
	// Initialize googleOauthConfig
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s/login/callback", url),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
}

const oauthGoogleUrlAPI = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get the session from the request
	session, err := store.Get(r, "session-name")
	if err != nil {
		log.Printf("Google Logout Error: %v", err)
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Clear session variables
	session.Values["userinfo"] = nil
	session.Values["oauthstate"] = nil

	// Save the session to apply changes
	if err := session.Save(r, w); err != nil {
		log.Printf("Google Logout Error: %v", err)
		http.Error(w, "Failed to save session logout.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func SetupCallbacks(mux *http.ServeMux) {
	mux.HandleFunc("/login", oauthGoogleLogin)
	mux.HandleFunc("/login/callback", oauthGoogleCallback)
	mux.HandleFunc("/logout", logoutHandler)
}

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	oauthState, err := generateStateOauthCookie(w, r)
	if err != nil {
		log.Println("Google Login Error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	/*
		AuthCodeURL receive state that is a token to protect the user from CSRF attacks. You must always provide a non-empty string and
		validate that it matches the the state query parameter on your redirect callback.
	*/
	u := googleOauthConfig.AuthCodeURL(oauthState)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Read oauthState from Cookie
	session, _ := store.Get(r, "session-name")
	// oauthState, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != session.Values["oauthstate"] {
		log.Println("Error: invalid oauth google state")
		// log.Println(err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data, err := getUserDataFromGoogle(r.FormValue("code"))
	if err != nil {
		log.Println(err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// save userinfo into the session
	session.Values["userinfo"] = data
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter, r *http.Request) (string, error) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		return "", fmt.Errorf("unable get session: %s", err.Error())
	}

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	session.Values["oauthstate"] = state
	err = session.Save(r, w)
	if err != nil {
		return "", fmt.Errorf("unable save oauth state to session: %s", err.Error())
	}

	return state, nil
}

func getUserDataFromGoogle(code string) ([]byte, error) {
	// Use code to get token and get user info from Google.

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %s", err.Error())
	}
	response, err := http.Get(oauthGoogleUrlAPI + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %s", err.Error())
	}
	return contents, nil
}

func GetUserDataFromSession(r *http.Request) (User, error) {
	// Get the session from the request
	session, err := store.Get(r, "session-name")
	if err != nil {
		return User{}, err
	}

	// Check if userinfo is stored in the session
	if userinfo, ok := session.Values["userinfo"].([]byte); ok {
		// Parse userinfo into a User struct
		// log.Printf("Values: %s", session.Values["userinfo"])
		var user User
		err := json.Unmarshal(userinfo, &user)
		if err != nil {
			return User{}, err
		}
		return user, nil
	}

	return User{}, errors.New("user data not found in session")
}

func IsUserAuthenticated(r *http.Request) bool {
	// Get the session from the request
	session, err := store.Get(r, "session-name")
	if err != nil {
		return false
	}

	// Check if userinfo is stored in the session
	userinfo, ok := session.Values["userinfo"].([]byte)
	if !ok {
		return false
	}

	var user User
	err = json.Unmarshal(userinfo, &user)
	if err != nil {
		return false
	}
	return user.VerifiedEmail
}
