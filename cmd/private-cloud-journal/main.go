package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"private-cloud-journal/internal/auth"
	"private-cloud-journal/internal/ui"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Secure Drive Notes (JSON Privacy)")
	myWindow.Resize(fyne.NewSize(950, 650))

	if auth.IsFirstRun() {
		ui.ShowRegistrationScreen(myWindow)
	} else {
		ui.ShowLoginScreen(myWindow)
	}

	myWindow.ShowAndRun()
}
