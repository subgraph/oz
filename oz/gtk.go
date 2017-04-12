package main

import (
	"fmt"
	"os"

	"github.com/gotk3/gotk3/gtk"
)

func promptConfirmShell(chanb chan bool, sandbox string) {
	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		fmt.Printf("Unable to create window: %v\n", err)
		os.Exit(1)
	}
	win.SetTitle("OZ Launch Shell: " + sandbox)
	win.SetModal(true)
	win.SetKeepAbove(true)
	win.SetDecorated(true)
	win.SetUrgencyHint(true)
	win.SetDeletable(false)
	win.SetResizable(false)
	win.SetIconName("dialog-question")

	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	headerbar, err := gtk.HeaderBarNew()
	if err != nil {
		fmt.Printf("Unable to create headerbar: %v\n", err)
		os.Exit(1)
	}
	headerbar.SetTitle("OZ Launch Shell")
	headerbar.SetSubtitle(sandbox)
	headerbar.SetShowCloseButton(false)

	win.SetTitlebar(headerbar)

	win.Add(promptWindowWidget(chanb, sandbox, win))

	win.ShowAll()
	gtk.Main()

	chanb <- false
}

func promptWindowWidget(chanb chan bool, sandbox string, win *gtk.Window) *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create grid: %v\n", err)
		os.Exit(1)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	topMsg := "Do you really want to launch a shell in:"
	topLabel, err := gtk.LabelNew(topMsg)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}
	topLabel.SetMarkup("<b>" + topMsg + "</b>")

	nameLabel, err := gtk.LabelNew(sandbox)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}
	nameLabel.SetMarkup("<u>" + sandbox + "</u>")

	btnGrid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create btnGrid: %v\n", err)
		os.Exit(1)
	}
	btnGrid.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
	btnGrid.SetColumnHomogeneous(true)

	btnCancel, err := gtk.ButtonNewWithLabel("Cancel")
	if err != nil {
		fmt.Printf("Unable to create btnCancel: %v\n", err)
		os.Exit(1)
	}
	//btnCancel.SetHAlign(gtk.ALIGN_START)

	btnYes, err := gtk.ButtonNewWithLabel("Yes")
	if err != nil {
		fmt.Printf("Unable to create btnYes: %v\n", err)
		os.Exit(1)
	}

	btnCancel.Connect("clicked", win.Destroy)
	btnYes.Connect("clicked", func() {
		chanb <- true
		win.Destroy()
	})
	//btnYes.SetHAlign(gtk.ALIGN_END)

	btnGrid.Add(btnCancel)
	btnGrid.Add(btnYes)
	btnGrid.SetColumnSpacing(25)

	grid.SetRowSpacing(25)
	grid.Container.Widget.SetMarginStart(15)
	grid.Container.Widget.SetMarginEnd(15)
	grid.Container.Widget.SetMarginTop(15)
	grid.Container.Widget.SetMarginBottom(15)
	grid.Add(topLabel)
	grid.Add(nameLabel)
	grid.Add(btnGrid)

	topLabel.SetHExpand(true)
	nameLabel.SetHExpand(true)
	btnGrid.SetHExpand(true)

	return &grid.Container.Widget
}
