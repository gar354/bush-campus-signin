package spreadsheetAPI

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var googleOauthConfig *oauth2.Config

func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize googleOauthConfig
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		ClientID:     os.Getenv("GOOGLE_SPREADSHEET_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_SPREADSHEET_OAUTH_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/spreadsheets"},
		Endpoint:     google.Endpoint,
	}
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "creds/token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// SubmitSpreadSheetData submits data to the spreadsheet.
func SubmitSpreadSheetData(email, signinType string) error {
	ctx := context.Background()

	// Authenticate and get the Sheets client
	client := getClient(googleOauthConfig)
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	// Get the current date to label the tab
	currentDate := time.Now().Format("2006-01-02")

	// Prepare the data to be appended
	values := [][]interface{}{
		{email, time.Now().Format("2006-01-02 15:04:05"), signinType},
	}

	// Define the range to append data
	rangeToAppend := fmt.Sprintf("%s!A:E", currentDate)

	// Create the request body
	rb := &sheets.ValueRange{
		Values: values,
	}

	// Append the data to the spreadsheet
	_, err = srv.Spreadsheets.Values.Append(os.Getenv("GOOGLE_SPREADSHEET_ID"), rangeToAppend, rb).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	return nil
}

