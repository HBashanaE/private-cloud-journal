package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"private-cloud-journal/internal/auth"
	"private-cloud-journal/internal/ui"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Private Cloud Journal")
	myWindow.Resize(fyne.NewSize(950, 650))

	if auth.IsFirstRun() {
		ui.ShowRegistrationScreen(myWindow)
	} else {
		ui.ShowLoginScreen(myWindow)
	}

	myWindow.ShowAndRun()
}
