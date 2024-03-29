package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/jesses-code-adventures/excavator/audio"
	"github.com/jesses-code-adventures/excavator/core"
	"github.com/jesses-code-adventures/excavator/server"
	"github.com/jesses-code-adventures/excavator/window"

	// Frontend
	tea "github.com/charmbracelet/bubbletea"
)

// ////////////////////// APP ////////////////////////
type App struct {
	server         *server.Server
	bubbleTeaModel window.Model
	logFile        *os.File
}

// Construct the app
func NewApp(cliFlags *server.Flags) App {
	logFilePath := filepath.Join(cliFlags.Data, cliFlags.LogFile)
	f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file in NewApp: %v", err)
	}
	log.SetOutput(f)
	audioPlayer := audio.NewAudioPlayer()
	server, err1 := server.NewServer(audioPlayer, cliFlags)
	server, err = server.AddUserAndRoot()
	needsUserAndRoot := false
	if err != nil || err1 != nil {
		needsUserAndRoot = true
	}
	return App{
		server:         &server,
		bubbleTeaModel: window.ExcavatorModel(&server, needsUserAndRoot),
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
		select {}
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
