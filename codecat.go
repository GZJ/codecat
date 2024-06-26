package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
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
	reader      io.RuneReader
	Interval    int
	Color       string
	SoundPlayer interface {
		Play()
	}
}

func NewPrinter(reader io.RuneReader) *Printer {
	return &Printer{
		reader: reader,
	}
}

func (p *Printer) Print() error {
	color := Palette[p.Color]

	for {
		rn, _, err := p.reader.ReadRune()
		if err != nil {
			break
		}

		fmt.Printf(color, string(rn))
		if rn != ' ' && rn != '\t' && rn != '\n' && rn != '\r' {
			p.SoundPlayer.Play()
		}
		time.Sleep(time.Duration(p.Interval) * time.Millisecond)
	}

	return nil
}

//go:embed sfx.wav
var sfx []byte

//go:embed codecat.go
var self []byte

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

	flag.StringVar(&codeFile, "code-file", "", "Path to code file")
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
		sfxReadCloser io.ReadCloser
		err           error
	)
	if soundFile == "" {
		reader := bytes.NewReader(sfx)
		sfxReadCloser = io.NopCloser(reader)
	} else {
		sfxReadCloser, err = os.Open(soundFile)
		if err != nil {
			log.Fatal("Error opening file:", err)
			return
		}
	}

	soundPlayer, err := NewSoundPlayer(sfxReadCloser)
	if err != nil {
		log.Fatal(err)
	}
	defer soundPlayer.Close()

	//printer
	var (
		r              *bufio.Reader
		selfReadCloser io.ReadCloser
	)

	if !isPipe(os.Stdin) {
		r = bufio.NewReader(os.Stdin)
	} else {
		if codeFile == "" {
			reader := bytes.NewReader(self)
			selfReadCloser = io.NopCloser(reader)
			r = bufio.NewReader(selfReadCloser)
		} else {
			file := codeFile
			f, err := os.Open(file)
			if err != nil {
				log.Fatal("Error opening file:", err)
				return
			}
			defer f.Close()
			r = bufio.NewReader(f)
		}
	}

	printer := NewPrinter(r)
	printer.SoundPlayer = soundPlayer
	printer.Color = color
	printer.Interval = interval
	err = printer.Print()
	if err != nil {
		log.Fatal("Error printing content:", err)
	}
}

func isPipe(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
