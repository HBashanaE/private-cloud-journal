package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

var srv *drive.Service
var appFolderID string

// --- AUTH & SETUP (Same as before) ---

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

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

func saveToken(path string, token *oauth2.Token) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func InitDriveService() error {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		return fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err = drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	// Setup Folder
	appFolderID, err = GetOrCreateFolder("MyNotes")
	if err != nil {
		return fmt.Errorf("unable to setup app folder: %v", err)
	}

	return nil
}

func GetOrCreateFolder(folderName string) (string, error) {
	q := fmt.Sprintf("mimeType='application/vnd.google-apps.folder' and name='%s' and trashed=false", folderName)
	r, err := srv.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return "", err
	}
	if len(r.Files) > 0 {
		return r.Files[0].Id, nil
	}
	f := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	file, err := srv.Files.Create(f).Fields("id").Do()
	if err != nil {
		return "", err
	}
	return file.Id, nil
}

// --- NEW LOGIC FOR JSON & PRIVACY ---

// GenerateRandomID creates a random string for the filename
func GenerateRandomID() string {
	bytes := make([]byte, 8) // 16 characters hex
	if _, err := rand.Read(bytes); err != nil {
		return "unknown_id"
	}
	return hex.EncodeToString(bytes)
}

// DriveFile represents the file metadata on Drive
type DriveFile struct {
	DriveID string // Google Drive ID (e.g., 1A2B...)
	Name    string // Filename (e.g., a4f9...json)
}

// ListDriveFiles returns all the raw files in our folder.
// Note: It does NOT return the Titles anymore, because Titles are encrypted inside!
func ListDriveFiles() ([]DriveFile, error) {
	q := fmt.Sprintf("'%s' in parents and mimeType != 'application/vnd.google-apps.folder' and trashed = false", appFolderID)

	// We list all files.
	r, err := srv.Files.List().Q(q).Fields("files(id, name)").Do()
	if err != nil {
		return nil, err
	}

	var files []DriveFile
	for _, f := range r.Files {
		files = append(files, DriveFile{DriveID: f.Id, Name: f.Name})
	}
	return files, nil
}

// SaveJSONNote encrypts and uploads the JSON struct
func SaveJSONNote(driveID string, data NoteData, password string) (string, error) {
	// 1. Marshal struct to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// 2. Encrypt the JSON string
	encryptedJSON, err := Encrypt(password, string(jsonBytes))
	if err != nil {
		return "", err
	}

	// 3. Prepare Content
	contentReader := strings.NewReader(encryptedJSON)

	// 4. Determine Filename (Use the ID from the struct)
	filename := data.ID + ".json"

	fileMetadata := &drive.File{
		Name: filename,
	}

	if driveID == "" {
		// CREATE NEW
		fileMetadata.Parents = []string{appFolderID}
		f, err := srv.Files.Create(fileMetadata).Media(contentReader).Fields("id").Do()
		if err != nil {
			return "", err
		}
		return f.Id, nil
	} else {
		// UPDATE EXISTING (We only update content, filename usually stays same)
		_, err := srv.Files.Update(driveID, fileMetadata).Media(contentReader).Do()
		if err != nil {
			return "", err
		}
		return driveID, nil
	}
}

// FetchAndDecryptNote downloads a file, decrypts it, and unmarshals it into NoteData
func FetchAndDecryptNote(driveID string, password string) (*NoteData, error) {
	// 1. Download Content
	resp, err := srv.Files.Get(driveID).Download()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	encryptedBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 2. Decrypt
	jsonStr, err := Decrypt(password, string(encryptedBytes))
	if err != nil {
		return nil, err
	}

	// 3. Unmarshal
	var note NoteData
	if err := json.Unmarshal([]byte(jsonStr), &note); err != nil {
		return nil, err
	}

	return &note, nil
}
