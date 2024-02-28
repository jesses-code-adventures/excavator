package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/jesses-code-adventures/excavator/audio"
	"github.com/jesses-code-adventures/excavator/core"
	"github.com/jesses-code-adventures/excavator/server"
	"github.com/jesses-code-adventures/excavator/ui"

	// Frontend
	tea "github.com/charmbracelet/bubbletea"
)

// ////////////////////// APP ////////////////////////
type App struct {
	server         *server.Server
	bubbleTeaModel ui.Model
	logFile        *os.File
}

// Construct the app
func NewApp(cliFlags *server.Flags) App {
	logFilePath := filepath.Join(cliFlags.Data, cliFlags.LogFile)
	f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	audioPlayer := audio.NewAudioPlayer()
	server := server.NewServer(audioPlayer, cliFlags)
	return App{
		server:         server,
		bubbleTeaModel: ui.ExcavatorModel(server),
		logFile:        f,
	}
}

// chris_brown_run_it.ogg
func main() {
	cliFlags := server.ParseFlags()
	core.CreateDirectories(cliFlags.Data)
	logFilePath := filepath.Join(cliFlags.Data, cliFlags.LogFile)
	if cliFlags.Watch {
		core.Watch(logFilePath, 10)
	} else {
		app := NewApp(cliFlags)
		defer app.logFile.Close()
		defer app.server.Player.Close()
		defer app.server.Db.Close()
		p := tea.NewProgram(
			app.bubbleTeaModel,
			tea.WithAltScreen(),
		)
		_, err := p.Run()
		if err != nil {
			log.Fatalf("Failed to run program: %v", err)
		}
	}
}
