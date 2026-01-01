package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

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
var currentNoteID string  // Keeps track if we are editing an existing note or creating a new one
var noteCache []DriveNote // Stores the list of notes from Drive

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Secure Drive Notes")
	myWindow.Resize(fyne.NewSize(900, 600))

	if IsFirstRun() {
		showRegistrationScreen(myWindow)
	} else {
		showLoginScreen(myWindow)
	}

	myWindow.ShowAndRun()
}

// --- Screen 1: Registration ---
func showRegistrationScreen(w fyne.Window) {
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Create Master Password")
	confirmEntry := widget.NewPasswordEntry()
	confirmEntry.SetPlaceHolder("Confirm Password")
	errorLabel := widget.NewLabel("")

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
		// Initialize Drive after registration
		initDriveAndShowApp(w)
	})

	content := container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Setup Secure Notes", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			passEntry, confirmEntry, registerBtn, errorLabel,
		),
	)
	w.SetContent(content)
}

// --- Screen 2: Login ---
func showLoginScreen(w fyne.Window) {
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Enter Master Password")
	errorLabel := widget.NewLabel("")

	loginBtn := widget.NewButton("Unlock", func() {
		if VerifyPassword(passEntry.Text) {
			sessionPassword = passEntry.Text
			initDriveAndShowApp(w)
		} else {
			errorLabel.SetText("Incorrect Password")
			passEntry.SetText("")
		}
	})

	content := container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Welcome Back", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			passEntry, loginBtn, errorLabel,
		),
	)
	w.SetContent(content)
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

	// 1. Sidebar List
	ui.NoteList = widget.NewList(
		func() int { return len(noteCache) },
		func() fyne.CanvasObject { return widget.NewLabel("Template Note Title") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(noteCache[i].Name)
		},
	)

	// Handle Note Selection
	ui.NoteList.OnSelected = func(id widget.ListItemID) {
		selectedNote := noteCache[id]
		currentNoteID = selectedNote.ID
		ui.TitleEntry.SetText(selectedNote.Name)
		ui.StatusLabel.SetText("Downloading...")

		// Fetch and Decrypt in background
		go func() {
			// 1. Download
			encryptedContent, err := DownloadNoteContent(selectedNote.ID)
			if err != nil {
				ui.StatusLabel.SetText("Download Error: " + err.Error())
				return
			}

			// 2. Decrypt
			plainText, err := Decrypt(sessionPassword, encryptedContent)
			if err != nil {
				// If decryption fails, it might be a plain text file or wrong password
				ui.StatusLabel.SetText("Decryption Error: " + err.Error())
				return
			}

			// 3. Update UI
			ui.BodyEntry.SetText(plainText)
			ui.StatusLabel.SetText("Loaded: " + selectedNote.Name)
		}()
	}

	// 2. Editor Area
	ui.TitleEntry = widget.NewEntry()
	ui.TitleEntry.SetPlaceHolder("Note Topic / Title")

	ui.BodyEntry = widget.NewMultiLineEntry()
	ui.BodyEntry.SetPlaceHolder("Write your secure notes here...")
	ui.BodyEntry.Wrapping = fyne.TextWrapWord

	ui.StatusLabel = widget.NewLabel("Ready. Connected to Google Drive.")

	// 3. Buttons

	// NEW NOTE BUTTON: Clears the state so we can save a fresh file
	ui.NewBtn = widget.NewButton("New Note", func() {
		currentNoteID = "" // Reset ID to empty -> triggers "Create" logic in Drive
		ui.TitleEntry.SetText("")
		ui.BodyEntry.SetText("")
		ui.StatusLabel.SetText("New note started.")
		ui.NoteList.UnselectAll() // Visually deselect the list
	})

	ui.SaveBtn = widget.NewButton("Encrypt & Save Cloud", func() {
		title := ui.TitleEntry.Text
		body := ui.BodyEntry.Text

		if title == "" || body == "" {
			ui.StatusLabel.SetText("Error: Title and Body required")
			return
		}

		ui.StatusLabel.SetText("Encrypting...")

		// Encrypt
		encryptedData, err := Encrypt(sessionPassword, body)
		if err != nil {
			ui.StatusLabel.SetText("Encryption failed: " + err.Error())
			return
		}

		ui.StatusLabel.SetText("Uploading...")

		// Upload to Drive (Background)
		go func() {
			newID, err := SaveNote(currentNoteID, title, encryptedData)
			if err != nil {
				ui.StatusLabel.SetText("Upload failed: " + err.Error())
				return
			}
			currentNoteID = newID // Update ID if it was a new note
			ui.StatusLabel.SetText("Saved successfully!")
			refreshNotes(ui) // Refresh list to see new file
		}()
	})

	ui.RefreshBtn = widget.NewButton("Refresh List", func() {
		refreshNotes(ui)
	})

	// Layout
	topBar := container.NewHBox(widget.NewLabel("Topic:"), ui.TitleEntry, layout.NewSpacer(), ui.NewBtn, ui.RefreshBtn)

	editorContent := container.NewBorder(
		topBar,
		container.NewVBox(ui.StatusLabel, ui.SaveBtn),
		nil, nil, ui.BodyEntry,
	)

	split := container.NewHSplit(
		container.New(layout.NewMaxLayout(), ui.NoteList),
		editorContent,
	)
	split.SetOffset(0.3)

	w.SetContent(split)

	// Initial Load
	refreshNotes(ui)
}

func refreshNotes(ui *UIComponents) {
	go func() {
		notes, err := ListNotes()
		if err != nil {
			log.Println("Error listing notes:", err)
			return
		}
		noteCache = notes
		ui.NoteList.Refresh()
	}()
}
