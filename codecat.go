package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
	flag "github.com/spf13/pflag"
)

var Palette = map[string]string{
	"black":         "\033[30m%s\033[0m",
	"red":           "\033[31m%s\033[0m",
	"green":         "\033[32m%s\033[0m",
	"yellow":        "\033[33m%s\033[0m",
	"blue":          "\033[34m%s\033[0m",
	"magenta":       "\033[35m%s\033[0m",
	"cyan":          "\033[36m%s\033[0m",
	"white":         "\033[37m%s\033[0m",
	"brightblack":   "\033[90m%s\033[0m",
	"brightred":     "\033[91m%s\033[0m",
	"brightgreen":   "\033[92m%s\033[0m",
	"brightyellow":  "\033[93m%s\033[0m",
	"brightblue":    "\033[94m%s\033[0m",
	"brightmagenta": "\033[95m%s\033[0m",
	"brightcyan":    "\033[96m%s\033[0m",
	"brightwhite":   "\033[97m%s\033[0m",
}

type Printer struct {
	reader      io.Reader
	Interval    int
	Color       string
	SoundPlayer interface {
		Play()
	}
}

func NewPrinter(reader io.Reader) *Printer {
	return &Printer{
		reader: reader,
	}
}

func (p *Printer) Print() error {
	content, err := io.ReadAll(p.reader)
	if err != nil {
		return err
	}

	color := Palette[p.Color]

	for _, char := range string(content) {
		fmt.Printf(color, string(char))
		if char != ' ' && char != '\t' && char != '\n' && char != '\r' {
			p.SoundPlayer.Play()
		}
		time.Sleep(time.Duration(p.Interval) * time.Millisecond)
	}

	return nil
}

//go:embed sfx.wav
var sfx []byte

type SoundPlayer struct {
	reader   io.ReadCloser
	streamer beep.StreamSeekCloser
	buffer   *beep.Buffer
}

func NewSoundPlayer(r io.ReadCloser) (*SoundPlayer, error) {
	streamer, format, err := wav.Decode(r)
	if err != nil {
		log.Fatal(err)
	}
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)

	return &SoundPlayer{
		reader:   r,
		streamer: streamer,
		buffer:   buffer,
	}, nil
}

func (sp *SoundPlayer) Play() {
	s := sp.buffer.Streamer(0, sp.buffer.Len())
	speaker.Play(s)
}

func (sp *SoundPlayer) Close() {
	sp.streamer.Close()
	sp.reader.Close()
}

// main
func main() {
	var (
		codeFile  string
		soundFile string
		color     string
		interval  int
	)

	flag.StringVar(&codeFile, "code-file", "codecat.go", "Path to code file")
	flag.StringVar(&soundFile, "sound-file", "", "Path to sound file")
	flag.StringVar(&color, "color", "green", "Print color")
	flag.IntVar(&interval, "interval", 50, "Print interval (ms)")

	flag.Parse()

	a := flag.Args()
	if len(a) > 0 {
		codeFile = a[0]
	}

	//soundPlaer
	var (
		readCloser io.ReadCloser
		err        error
	)
	if soundFile == "" {
		reader := bytes.NewReader(sfx)
		readCloser = io.NopCloser(reader)
	} else {
		readCloser, err = os.Open(soundFile)
		if err != nil {
			log.Fatal("Error opening file:", err)
			return
		}
	}

	soundPlayer, err := NewSoundPlayer(readCloser)
	if err != nil {
		log.Fatal(err)
	}
	defer soundPlayer.Close()

	//printer
	file := codeFile
	f, err := os.Open(file)
	if err != nil {
		log.Fatal("Error opening file:", err)
		return
	}
	defer f.Close()

	printer := NewPrinter(f)
	printer.SoundPlayer = soundPlayer
	printer.Color = color
	printer.Interval = interval
	err = printer.Print()
	if err != nil {
		log.Fatal("Error printing content:", err)
	}

}
