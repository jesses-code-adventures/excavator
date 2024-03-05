package audio

import (
	"log"
	"os"
	"path/filepath"
	"time"
	// Audio
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

// ////////////////////// AUDIO HANDLING ////////////////////////

// Audio file type enum
type AudioFile int

// Audio file type enum values
const (
	MP3 AudioFile = iota
	WAV
	FLAC
)

// String representation of an audio file type
func (a *AudioFile) String() string {
	return [...]string{"mp3", "wav", "flac"}[*a]
}

// Construct an audio file type from a string
func (a *AudioFile) FromExtension(s string) {
	switch s {
	case ".mp3":
		*a = MP3
	case ".wav":
		*a = WAV
	case ".flac":
		*a = FLAC
	default:
		log.Fatalf("Unsupported audio file type: %v", s)
	}
}

// Audio player struct
type Player struct {
	Format      beep.Format
	Streamer    beep.StreamSeekCloser
	Commands    chan string
	Playing     bool
	NextCommand *string
}

// Push a play command to the audio player's commands channel
func (a *Player) pushPlayCommand(path string) {
	a.NextCommand = &path
	a.Commands <- path
}

// Construct the audio player
func NewAudioPlayer() *Player {
	sampleRate := beep.SampleRate(48000)
	format := beep.Format{SampleRate: sampleRate, NumChannels: 2, Precision: 4}
	player := Player{
		Format:   format,
		Playing:  false,
		Commands: make(chan string),
	}
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	go func() {
		player.Run()
	}()
	return &player
}

// Close the audio player
func (a *Player) Close() {
	speaker.Lock()
	if a.Streamer != nil {
		a.Streamer.Close()
	}
	speaker.Unlock()
	speaker.Close()
}

// Get a streamer which will buffer playback of one file
func (a *Player) GetStreamer(path string, f *os.File) (beep.StreamSeekCloser, beep.Format, error) {
	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error
	switch filepath.Ext(path) {
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	case ".flac":
		streamer, format, err = flac.Decode(f)
	}
	if err != nil {
		log.Print(err)
		return nil, format, err
	}
	return streamer, format, nil
}

// Close the current streamer
func (a *Player) CloseStreamer() {
	if a.Streamer != nil {
		a.Streamer.Close()
	}
	a.Streamer = nil
}

// Handle a play command arriving in the audio player's commands channel
func (a *Player) handlePlayCommand(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("error opening file in handleplaycommand ", err)
	}
	defer f.Close()
	a.Playing = true
	streamer, format, err := a.GetStreamer(path, f)
	if err != nil {
		log.Printf("Failed to get streamer: %v", err)
		return
	}
	log.Printf("Playing file: \n--> path %s\n--> format%v", path, format)
	a.Streamer = streamer
	defer a.CloseStreamer()
	resampled := beep.Resample(4, format.SampleRate, a.Format.SampleRate, streamer)
	done := make(chan bool)
	speaker.Play(beep.Seq(resampled, beep.Callback(func() {
		a.Playing = false
		if a.NextCommand != nil && *a.NextCommand == path {
			a.NextCommand = nil
		}
		done <- true
	})))
	<-done
}

// Run the audio player, feeding it paths as play commands
func (a *Player) Run() {
	for {
		select {
		case path := <-a.Commands:
			if a.NextCommand != nil && *a.NextCommand != path {
				continue
			}
			a.handlePlayCommand(path)
		}
	}
}

// Play one audio file. If another file is already playing, close the current streamer and play the new file.
func (a *Player) PlayAudioFile(path string) {
	if a.Playing {
		// Close current streamer with any necessary cleanup
		a.CloseStreamer()
	}
	a.pushPlayCommand(path)
}
