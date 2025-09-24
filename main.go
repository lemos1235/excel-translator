package main

import (
	"exceltranslator/core"
	"exceltranslator/gui2"
)

func main() {
	gui2.CreateGUI(core.ProcessFile)
}
