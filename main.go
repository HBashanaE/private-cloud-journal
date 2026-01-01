package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// UIComponents holds references to widgets we need to update dynamically
type UIComponents struct {
	NoteList    *widget.List
	TitleEntry  *widget.Entry
	BodyEntry   *widget.Entry
	StatusLabel *widget.Label
	SaveBtn     *widget.Button
}

func main() {
	// 1. Initialize the App
	myApp := app.New()
	myWindow := myApp.NewWindow("Private Cloud Journal")

	// 2. Set initial window size
	myWindow.Resize(fyne.NewSize(800, 600))

	// 3. Create UI Components
	ui := &UIComponents{}

	// --- Left Sidebar (Note List) ---
	// Mock data for now
	data := []string{"Note 1: Ideas", "Note 2: Todo", "Note 3: Secrets"}

	ui.NoteList = widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Object")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(data[i])
		},
	)

	ui.NoteList.OnSelected = func(id widget.ListItemID) {
		ui.StatusLabel.SetText("Selected: " + data[id])
	}

	// --- Right Content Area (Editor) ---
	ui.TitleEntry = widget.NewEntry()
	ui.TitleEntry.SetPlaceHolder("Note Topic / Title")

	// This constructor configures the Entry to accept multiple lines
	ui.BodyEntry = widget.NewMultiLineEntry()
	ui.BodyEntry.SetPlaceHolder("Enter your secure notes here...")
	ui.BodyEntry.Wrapping = fyne.TextWrapWord

	ui.StatusLabel = widget.NewLabel("Ready")

	ui.SaveBtn = widget.NewButton("Encrypt & Save to Drive", func() {
		log.Println("Save button clicked")
		ui.StatusLabel.SetText("Saving...")
		//wait some time to simulate saving
		go func() {
			// Simulate a save delay
			time.Sleep(2 * time.Second)
			ui.StatusLabel.SetText("Saved successfully!")
		}()
	})

	// Layout for the Editor
	editorContent := container.NewBorder(
		container.NewVBox(widget.NewLabel("Topic:"), ui.TitleEntry), // Top
		container.NewVBox(ui.StatusLabel, ui.SaveBtn),               // Bottom
		nil,          // Left
		nil,          // Right
		ui.BodyEntry, // Center
	)

	// --- Split Container ---
	split := container.NewHSplit(
		container.New(layout.NewStackLayout(), ui.NoteList),
		editorContent,
	)
	split.SetOffset(0.3)

	// 4. Set Content and Run
	myWindow.SetContent(split)
	myWindow.ShowAndRun()
}
