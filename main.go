package main

import (
	"image/color"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NoteData is the structure we save inside the encrypted JSON
type NoteData struct {
	ID        string    `json:"id"` // Internal App ID (random string)
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AppNote holds the data + the Google Drive File ID needed for updates
type AppNote struct {
	DriveID string
	Data    NoteData
}

type UIComponents struct {
	NoteList    *widget.List
	TitleEntry  *widget.Entry
	BodyEntry   *widget.Entry
	StatusLabel *widget.Label
	SaveBtn     *widget.Button
	RefreshBtn  *widget.Button
	NewBtn      *widget.Button
}

// Global state
var sessionPassword string
var currentDriveID string // The Google Drive ID of the currently selected note
var currentAppID string   // The Internal App ID (filename)
var noteCache []AppNote   // Stores the fully decrypted notes for the list

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Secure Drive Notes (JSON Privacy)")
	myWindow.Resize(fyne.NewSize(950, 650))

	if IsFirstRun() {
		showRegistrationScreen(myWindow)
	} else {
		showLoginScreen(myWindow)
	}

	myWindow.ShowAndRun()
}

// --- Screen 1: Registration (First Run) ---
func showRegistrationScreen(w fyne.Window) {
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Create Master Password")

	confirmEntry := widget.NewPasswordEntry()
	confirmEntry.SetPlaceHolder("Confirm Password")

	errorLabel := widget.NewLabel("")
	errorLabel.Alignment = fyne.TextAlignCenter         // Center error text
	errorLabel.TextStyle = fyne.TextStyle{Italic: true} // Make it look like a status message

	registerBtn := widget.NewButton("Create Account", func() {
		if passEntry.Text == "" {
			errorLabel.SetText("Password cannot be empty")
			return
		}
		if passEntry.Text != confirmEntry.Text {
			errorLabel.SetText("Passwords do not match")
			return
		}
		if err := SavePasswordHash(passEntry.Text); err != nil {
			errorLabel.SetText("Error saving config: " + err.Error())
			return
		}
		sessionPassword = passEntry.Text
		initDriveAndShowApp(w)
	})
	registerBtn.Importance = widget.HighImportance // Makes the button visually prominent (usually blue)

	// --- CLEANER SPACER LOGIC ---
	// Create an invisible object that is 300 units wide and 0 units high.
	// This forces the parent container to be at least 300 units wide.
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(300, 0))

	// --- Layout Construction ---
	// We use a VBox for vertical stacking, but with added spacing items
	formContent := container.NewVBox(
		// Title
		widget.NewLabelWithStyle("Setup Secure Notes", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),

		// Spacing
		widget.NewLabel(""),

		// Inputs
		passEntry,
		confirmEntry,

		// Spacing
		widget.NewLabel(""),

		// Action
		registerBtn,

		// Status
		errorLabel,

		// Invisible spacer to force width
		container.NewHBox(layout.NewSpacer(), spacer, layout.NewSpacer()),
	)

	// Wrap in a Card for a nice background/border look
	card := widget.NewCard("", "", container.NewPadded(formContent))

	// Center the card on screen
	w.SetContent(container.NewCenter(card))
}

// --- Screen 2: Login (Subsequent Runs) ---
func showLoginScreen(w fyne.Window) {
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Enter Master Password")

	errorLabel := widget.NewLabel("")
	errorLabel.Alignment = fyne.TextAlignCenter
	errorLabel.TextStyle = fyne.TextStyle{Italic: true}

	loginBtn := widget.NewButton("Unlock Safe", func() {
		if VerifyPassword(passEntry.Text) {
			sessionPassword = passEntry.Text
			initDriveAndShowApp(w)
		} else {
			errorLabel.SetText("Incorrect Password")
			passEntry.SetText("")
		}
	})
	loginBtn.Importance = widget.HighImportance

	// --- CLEANER SPACER LOGIC ---
	// Create an invisible object that is 300 units wide and 0 units high.
	// This forces the parent container to be at least 300 units wide.
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(300, 0))

	// --- Layout Construction ---
	formContent := container.NewVBox(
		// Title
		widget.NewLabelWithStyle("Welcome Back", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),

		// Vertical Gap
		widget.NewLabel(""),

		// Input
		passEntry,

		// Vertical Gap
		widget.NewLabel(""),

		// Button
		loginBtn,

		// Error Message
		errorLabel,

		// Force width
		container.NewCenter(spacer),
	)

	// Wrap in Padded Card
	card := widget.NewCard("", "", container.NewPadded(formContent))

	w.SetContent(container.NewCenter(card))
}

// Helper to init drive inside the GUI flow
func initDriveAndShowApp(w fyne.Window) {
	// Show a loading screen while we connect to Google
	w.SetContent(container.NewCenter(widget.NewLabel("Connecting to Google Drive...\nCheck your terminal if it's the first time!")))

	// Do this in a goroutine so UI doesn't freeze, but for the AUTH step specifically,
	// we need to wait because we can't show the app without the service.
	// Since the auth might require terminal interaction, we run it directly here.

	err := InitDriveService()
	if err != nil {
		dialog.ShowError(err, w)
		return
	}

	showMainApp(w)
}

// --- Screen 3: Main App ---

func showMainApp(w fyne.Window) {
	ui := &UIComponents{}

	// --- 1. Sidebar List ---
	ui.NoteList = widget.NewList(
		func() int { return len(noteCache) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template Note")
			label.TextStyle = fyne.TextStyle{Bold: true}
			return container.NewPadded(label)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			// Display the Title from the decrypted JSON
			title := noteCache[i].Data.Title
			if title == "" {
				title = "Untitled Note"
			}
			o.(*fyne.Container).Objects[0].(*widget.Label).SetText(title)
		},
	)

	ui.NoteList.OnSelected = func(id widget.ListItemID) {
		selected := noteCache[id]
		currentDriveID = selected.DriveID
		currentAppID = selected.Data.ID

		// Populate UI directly from Cache (no need to download again!)
		ui.TitleEntry.SetText(selected.Data.Title)
		ui.BodyEntry.SetText(selected.Data.Body)
		ui.StatusLabel.SetText("Loaded: " + selected.Data.ID)
	}

	// --- 2. Top Bar ---
	ui.TitleEntry = widget.NewEntry()
	ui.TitleEntry.SetPlaceHolder("Enter Topic / Title")

	ui.NewBtn = widget.NewButtonWithIcon("New", theme.FileIcon(), func() {
		currentDriveID = ""
		currentAppID = ""
		ui.TitleEntry.SetText("")
		ui.BodyEntry.SetText("")
		ui.StatusLabel.SetText("New note started.")
		ui.NoteList.UnselectAll()
	})

	ui.RefreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		refreshNotes(ui)
	})

	toolbarRight := container.NewHBox(ui.NewBtn, ui.RefreshBtn)
	topBar := container.NewBorder(nil, nil, widget.NewLabel("Topic: "), toolbarRight, ui.TitleEntry)

	// --- 3. Body ---
	ui.BodyEntry = widget.NewMultiLineEntry()
	ui.BodyEntry.SetPlaceHolder("Write your secure notes here...")
	ui.BodyEntry.Wrapping = fyne.TextWrapWord
	bodyArea := container.NewPadded(ui.BodyEntry)

	// --- 4. Bottom Bar ---
	ui.StatusLabel = widget.NewLabel("Ready.")
	ui.StatusLabel.Alignment = fyne.TextAlignCenter
	ui.StatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	ui.SaveBtn = widget.NewButtonWithIcon("Encrypt & Save JSON", theme.DocumentSaveIcon(), func() {
		title := ui.TitleEntry.Text
		body := ui.BodyEntry.Text
		if title == "" {
			ui.StatusLabel.SetText("Title is required")
			return
		}

		ui.StatusLabel.SetText("Packaging & Encrypting...")

		// Generate ID if new
		if currentAppID == "" {
			currentAppID = GenerateRandomID()
		}

		// Create Struct
		note := NoteData{
			ID:        currentAppID,
			Title:     title,
			Body:      body,
			UpdatedAt: time.Now(),
		}
		if currentDriveID == "" {
			note.CreatedAt = time.Now()
		}

		// Save (Encrypts the whole struct)
		go func() {
			driveID, err := SaveJSONNote(currentDriveID, note, sessionPassword)
			if err != nil {
				ui.StatusLabel.SetText("Save failed: " + err.Error())
				return
			}
			currentDriveID = driveID
			ui.StatusLabel.SetText("Saved securely as " + currentAppID + ".json")
			refreshNotes(ui)
		}()
	})
	ui.SaveBtn.Importance = widget.HighImportance

	bottomBar := container.NewVBox(ui.StatusLabel, ui.SaveBtn)

	// --- Layout ---
	editorContent := container.NewBorder(container.NewPadded(topBar), container.NewPadded(bottomBar), nil, nil, bodyArea)
	split := container.NewHSplit(container.New(layout.NewMaxLayout(), ui.NoteList), editorContent)
	split.SetOffset(0.25)

	w.SetContent(split)

	// Initial Load
	refreshNotes(ui)
}

func refreshNotes(ui *UIComponents) {
	ui.StatusLabel.SetText("Syncing: Fetching file list...")

	go func() {
		// 1. List Files (Only gets IDs and Encrypted filenames)
		files, err := ListDriveFiles()
		if err != nil {
			log.Println("List error:", err)
			return
		}

		// 2. Download & Decrypt each file to build the cache
		// In a real app, we would parallelize this or use a local DB cache.
		var newCache []AppNote

		ui.StatusLabel.SetText("Syncing: Decrypting notes...")

		for _, f := range files {
			// Skip if not json
			if !strings.HasSuffix(f.Name, ".json") {
				continue
			}

			noteData, err := FetchAndDecryptNote(f.DriveID, sessionPassword)
			if err != nil {
				log.Println("Failed to decrypt file:", f.Name, err)
				continue
			}

			newCache = append(newCache, AppNote{
				DriveID: f.DriveID,
				Data:    *noteData,
			})
		}

		// 3. Update UI
		noteCache = newCache
		ui.NoteList.Refresh()
		ui.StatusLabel.SetText("Sync Complete.")
	}()
}
