package spreadsheetAPI

import (
	"context"
	"fmt"
	"time"
	"strings"

	"golang.org/x/oauth2/jwt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var keyFilePath string = "data/token.json"

var jwtConfig *jwt.Config
var spreadsheetID string

func SetupAuthConfig(email string, key string, id string) {
	key = strings.Replace(key,"\\n", "\n", -1) // fix escaping in env var
	jwtConfig = &jwt.Config{
		Email:      email,
		PrivateKey: []byte(key),
		TokenURL:   google.JWTTokenURL,
		Scopes:     []string{sheets.SpreadsheetsScope},
	}
	spreadsheetID = id
}

func SubmitSpreadSheetData(email string, signinType string, freePeriod string, reason string) error {
	// Create a context
	ctx := context.Background()

	// Configure the JWT token to use for authentication
	client := jwtConfig.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	// Get the current date to label the tab
	currentDate := time.Now().Format("2006-01-02")

	sheetExists := false
	resp, err := srv.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to get spreadsheet: %v", err)
	}
	for _, sheet := range resp.Sheets {
		if sheet.Properties.Title == currentDate {
			sheetExists = true
			break
		}
	}
	if !sheetExists {
		_, err := srv.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				{
					AddSheet: &sheets.AddSheetRequest{
						Properties: &sheets.SheetProperties{
							Title: currentDate,
						},
					},
				},
			},
		}).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("unable to create new sheet: %v", err)
		}
	}
	// add columns
	if !sheetExists {
		columns := [][]interface{}{
			{"Email", "Time", "Sign Out/Sign In", "Free Period?", "Reason"},
		}
		// Define the range to append data
		rangeToAppend := fmt.Sprintf("%s!A:E", currentDate)

		// Create the request body
		rb := &sheets.ValueRange{
			Values: columns,
		}

		// Append the data to the spreadsheet
		_, err = srv.Spreadsheets.Values.Append(spreadsheetID, rangeToAppend, rb).ValueInputOption("RAW").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("unable to append data to sheet: %v", err)
		}
	}

	// Prepare the data to be appended
	values := [][]interface{}{
		{email, time.Now().Format("2006-01-02 15:04:05"), signinType, freePeriod, reason},
	}

	// Define the range to append data
	rangeToAppend := fmt.Sprintf("%s!A:E", currentDate)

	// Create the request body
	requestBody := &sheets.ValueRange{
		Values: values,
	}

	// Append the data to the spreadsheet
	_, err = srv.Spreadsheets.Values.Append(spreadsheetID,
		rangeToAppend, requestBody).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	return nil
}
