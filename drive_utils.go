package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Global Drive Service
var srv *drive.Service

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first time.
	tokFile := "token.json"
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
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	// Since we are in a GUI app, let's try to open the browser automatically
	// (This works on Mac/Windows/Linux usually)
	// For now, we will print to console and ask user to paste back in terminal.
	// NOTE: In a polished app, you would pop up a dialog input box here.
	// For this MVP, look at your TERMINAL window for the link.

	var authCode string
	fmt.Print("Type the authorization code here: ")
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
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	return t, err
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

// --- APP LOGIC ---

// InitDriveService reads credentials.json and sets up the Drive client
func InitDriveService() error {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		return fmt.Errorf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err = drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}
	return nil
}

// DriveNote represents a file found in Drive
type DriveNote struct {
	ID   string
	Name string
}

// ListNotes finds all files that we have created (identified by a custom property or name)
func ListNotes() ([]DriveNote, error) {
	// We search for files not in trash, and make sure they are text files
	// To keep it simple, we just look for text/plain files.
	// In production, we would use `appProperties has { key='app' and value='secure_notes' }`
	q := "mimeType = 'text/plain' and trashed = false"

	r, err := srv.Files.List().Q(q).Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		return nil, err
	}

	var notes []DriveNote
	for _, f := range r.Files {
		// Filter: We only want notes that look like ours (e.g. they don't have extensions usually in our logic, or we just show all text files)
		notes = append(notes, DriveNote{ID: f.Id, Name: f.Name})
	}
	return notes, nil
}

// SaveNote uploads a new file or updates an existing one
// If id is empty, it creates a new file. If id is present, it updates that file.
func SaveNote(id, title, content string) (string, error) {
	fileMetadata := &drive.File{
		Name:     title,
		MimeType: "text/plain",
	}

	// Create a reader from the string content
	contentReader := strings.NewReader(content)

	if id == "" {
		// --- CREATE NEW ---
		// FIX: Use .Media() to attach content, instead of passing it as an argument
		f, err := srv.Files.Create(fileMetadata).Media(contentReader).Fields("id").Do()
		if err != nil {
			return "", err
		}
		return f.Id, nil
	} else {
		// --- UPDATE EXISTING ---
		// FIX: Use .Media() to attach content here as well
		_, err := srv.Files.Update(id, fileMetadata).Media(contentReader).Do()
		if err != nil {
			return "", err
		}
		return id, nil
	}
}

// DownloadNoteContent gets the text body of a file
func DownloadNoteContent(fileId string) (string, error) {
	// 1. Request the file content from Drive
	resp, err := srv.Files.Get(fileId).Download()
	if err != nil {
		return "", err
	}
	// Ensure the connection closes when we are done
	defer resp.Body.Close()

	// 2. Read all bytes from the response body
	contentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 3. Convert bytes to string and return
	return string(contentBytes), nil
}
