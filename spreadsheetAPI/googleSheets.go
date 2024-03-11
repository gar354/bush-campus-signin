package spreadsheetAPI

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var keyFilePath string = "data/token.json"

func init() {
	// ensure token.json exists when starting the program
	_, err := os.ReadFile(keyFilePath)
	if err != nil {
		log.Fatalf("Unable to read service account key file: %v", err)
	}
}

func SubmitSpreadSheetData(email string, signinType string, freePeriod string, reason string) error {
	keyFile, err := os.ReadFile(keyFilePath)
	if err != nil {
		log.Fatalf("Unable to read service account key file: %v", err)
	}

	// Parse the service account key file
	conf, err := google.JWTConfigFromJSON(keyFile, sheets.SpreadsheetsScope)
	if err != nil {
		log.Fatalf("failed to parse service account key file: %v", err)
	}

	ctx := context.Background()
	client := conf.Client(ctx)
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	// Get the current date to label the tab
	currentDate := time.Now().Format("2006-01-02")

	sheetExists := false
	resp, err := srv.Spreadsheets.Get(os.Getenv("GOOGLE_SPREADSHEET_ID")).Context(ctx).Do()
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
		_, err := srv.Spreadsheets.BatchUpdate(os.Getenv("GOOGLE_SPREADSHEET_ID"), &sheets.BatchUpdateSpreadsheetRequest{
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
		_, err = srv.Spreadsheets.Values.Append(os.Getenv("GOOGLE_SPREADSHEET_ID"), rangeToAppend, rb).ValueInputOption("RAW").Context(ctx).Do()
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
	_, err = srv.Spreadsheets.Values.Append(os.Getenv("GOOGLE_SPREADSHEET_ID"),
		rangeToAppend, requestBody).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	return nil
}
