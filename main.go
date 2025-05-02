package main

import (
	"exceltranslator/core"
	"exceltranslator/pkg/gui"
)

func main() {
	gui.CreateGUI(core.ProcessExcelFile)
}
