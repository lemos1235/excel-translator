package main

import (
	"exceltranslator/core"
	"exceltranslator/gui"
)

func main() {
	gui.CreateGUI(core.ProcessExcelFile)
}
