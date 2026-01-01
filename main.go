package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type UIComponents struct {
	NoteList    *widget.List
	TitleEntry  *widget.Entry
	BodyEntry   *widget.Entry
	StatusLabel *widget.Label
	SaveBtn     *widget.Button
}

var sessionPassword string

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Secure Drive Notes")
	myWindow.Resize(fyne.NewSize(800, 600))

	// CHECK: Is this the first time the user is running the app?
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

	registerBtn := widget.NewButton("Create Account", func() {
		if passEntry.Text == "" {
			errorLabel.SetText("Password cannot be empty")
			return
		}
		if passEntry.Text != confirmEntry.Text {
			errorLabel.SetText("Passwords do not match")
			return
		}

		// Save the hash to disk
		err := SavePasswordHash(passEntry.Text)
		if err != nil {
			errorLabel.SetText("Error saving config: " + err.Error())
			return
		}

		// Store in memory for immediate use
		sessionPassword = passEntry.Text

		// Move to App
		showMainApp(w)
	})

	content := container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Setup Secure Notes", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Create a permanent Master Password.\nDon't lose this; we cannot recover it."),
			passEntry,
			confirmEntry,
			registerBtn,
			errorLabel,
		),
	)
	w.SetContent(content)
}

// --- Screen 2: Login (Subsequent Runs) ---
func showLoginScreen(w fyne.Window) {
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Enter Master Password")

	errorLabel := widget.NewLabel("")

	loginBtn := widget.NewButton("Unlock", func() {
		// Verify against stored hash
		if VerifyPassword(passEntry.Text) {
			sessionPassword = passEntry.Text
			showMainApp(w)
		} else {
			errorLabel.SetText("Incorrect Password")
			passEntry.SetText("") // Clear input
		}
	})

	content := container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Welcome Back", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			passEntry,
			loginBtn,
			errorLabel,
		),
	)
	w.SetContent(content)
}

// --- Screen 3: Main App ---
func showMainApp(w fyne.Window) {
	ui := &UIComponents{}

	// --- Sidebar ---
	data := []string{"Note 1: Ideas", "Note 2: Todo"}
	ui.NoteList = widget.NewList(
		func() int { return len(data) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(data[i]) },
	)

	// --- Editor ---
	ui.TitleEntry = widget.NewEntry()
	ui.TitleEntry.SetPlaceHolder("Topic")

	ui.BodyEntry = widget.NewMultiLineEntry()
	ui.BodyEntry.SetPlaceHolder("Secure notes...")
	ui.BodyEntry.Wrapping = fyne.TextWrapWord

	ui.StatusLabel = widget.NewLabel("Ready")

	ui.SaveBtn = widget.NewButton("Encrypt & Save", func() {
		txt := ui.BodyEntry.Text
		if txt == "" {
			return
		}

		// Use the authenticated sessionPassword
		encrypted, err := Encrypt(sessionPassword, txt)
		if err != nil {
			ui.StatusLabel.SetText("Error: " + err.Error())
			return
		}

		log.Println("--- Encrypted Data ---")
		log.Println(encrypted)
		ui.StatusLabel.SetText("Encrypted! Check Terminal.")
		log.Println("--- End Encrypted Data ---")

		// For demonstration, immediately decrypt
		decrypted, err := Decrypt(sessionPassword, encrypted)
		if err != nil {
			ui.StatusLabel.SetText("Decryption Error: " + err.Error())
			return
		}
		log.Println("--- Decrypted Data ---")
		log.Println(decrypted)
		log.Println("--- End Decrypted Data ---")
	})

	editorContent := container.NewBorder(
		container.NewVBox(widget.NewLabel("Topic:"), ui.TitleEntry),
		container.NewVBox(ui.StatusLabel, ui.SaveBtn),
		nil, nil, ui.BodyEntry,
	)

	split := container.NewHSplit(
		container.New(layout.NewMaxLayout(), ui.NoteList),
		editorContent,
	)
	split.SetOffset(0.3)
	w.SetContent(split)
}
