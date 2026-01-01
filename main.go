package main

import (
	"log"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
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

	// --- 1. Sidebar List (With Padding) ---
	ui.NoteList = widget.NewList(
		func() int { return len(noteCache) },
		func() fyne.CanvasObject {
			// We wrap the text in a Padded container so list items breathe
			label := widget.NewLabel("Template Note Title")
			label.TextStyle = fyne.TextStyle{Bold: true}
			return container.NewPadded(label)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			// Because we wrapped it in Padded, we must unpack it to get the label
			// Structure: Padded Container -> [0] Label
			o.(*fyne.Container).Objects[0].(*widget.Label).SetText(noteCache[i].Name)
		},
	)

	// Handle Note Selection
	ui.NoteList.OnSelected = func(id widget.ListItemID) {
		selectedNote := noteCache[id]
		currentNoteID = selectedNote.ID
		ui.TitleEntry.SetText(selectedNote.Name)
		ui.StatusLabel.SetText("Downloading...")

		go func() {
			encryptedContent, err := DownloadNoteContent(selectedNote.ID)
			if err != nil {
				ui.StatusLabel.SetText("Download Error: " + err.Error())
				return
			}
			plainText, err := Decrypt(sessionPassword, encryptedContent)
			if err != nil {
				ui.StatusLabel.SetText("Decryption Error: " + err.Error())
				return
			}
			ui.BodyEntry.SetText(plainText)
			ui.StatusLabel.SetText("Loaded: " + selectedNote.Name)
		}()
	}

	// --- 2. Top Bar (Topic + Buttons) ---
	ui.TitleEntry = widget.NewEntry()
	ui.TitleEntry.SetPlaceHolder("Enter Topic / Title")

	ui.NewBtn = widget.NewButtonWithIcon("New", theme.FileIcon(), func() {
		currentNoteID = ""
		ui.TitleEntry.SetText("")
		ui.BodyEntry.SetText("")
		ui.StatusLabel.SetText("New note started.")
		ui.NoteList.UnselectAll()
	})

	ui.RefreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		refreshNotes(ui)
	})

	// Layout Logic:
	// We use a Border layout.
	// Left: "Topic" Label
	// Right: Buttons
	// Center: TitleEntry (This makes it EXPAND to fill the gap!)
	toolbarRight := container.NewHBox(ui.NewBtn, ui.RefreshBtn)
	topBar := container.NewBorder(
		nil, nil,
		widget.NewLabel("Topic: "), // Left
		toolbarRight,               // Right
		ui.TitleEntry,              // Center (Expands)
	)

	// --- 3. Main Editor Area ---
	ui.BodyEntry = widget.NewMultiLineEntry()
	ui.BodyEntry.SetPlaceHolder("Write your secure notes here...")
	ui.BodyEntry.Wrapping = fyne.TextWrapWord
	// Add padding around the text box so it doesn't touch the window edges
	bodyArea := container.NewPadded(ui.BodyEntry)

	// --- 4. Bottom Bar (Status + Save) ---
	ui.StatusLabel = widget.NewLabel("Ready. Connected to Google Drive.")
	ui.StatusLabel.Alignment = fyne.TextAlignCenter
	ui.StatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	ui.SaveBtn = widget.NewButtonWithIcon("Encrypt & Save Cloud", theme.DocumentSaveIcon(), func() {
		title := ui.TitleEntry.Text
		body := ui.BodyEntry.Text

		if title == "" || body == "" {
			ui.StatusLabel.SetText("Error: Title and Body required")
			return
		}

		ui.StatusLabel.SetText("Encrypting...")
		encryptedData, err := Encrypt(sessionPassword, body)
		if err != nil {
			ui.StatusLabel.SetText("Encryption failed: " + err.Error())
			return
		}

		ui.StatusLabel.SetText("Uploading...")
		go func() {
			newID, err := SaveNote(currentNoteID, title, encryptedData)
			if err != nil {
				ui.StatusLabel.SetText("Upload failed: " + err.Error())
				return
			}
			currentNoteID = newID
			ui.StatusLabel.SetText("Saved successfully!")
			refreshNotes(ui)
		}()
	})
	ui.SaveBtn.Importance = widget.HighImportance // Make button Blue (Primary)

	bottomBar := container.NewVBox(ui.StatusLabel, ui.SaveBtn)

	// --- 5. Assemble the Right Side ---
	editorContent := container.NewBorder(
		container.NewPadded(topBar),    // Top (with padding)
		container.NewPadded(bottomBar), // Bottom (with padding)
		nil, nil,
		bodyArea, // Center
	)

	// --- 6. Final Split ---
	split := container.NewHSplit(
		container.New(layout.NewMaxLayout(), ui.NoteList), // Sidebar
		editorContent, // Main Content
	)
	split.SetOffset(0.25) // Sidebar takes 25% of width

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
