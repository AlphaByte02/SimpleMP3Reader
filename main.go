package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"

	"github.com/eiannone/keyboard"
)

func main() {
	retcode := 0
	defer func() { os.Exit(retcode) }()

	progressbarGenericOption := []progressbar.Option{
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionShowCount(),
		// progressbar.OptionSetVisibility(false),
	}

	songListPath := os.Args[1:]
	if len(os.Args) < 2 {
		scanner := bufio.NewScanner(os.Stdin)

		var filepath string
		for len(filepath) == 0 {
			fmt.Print("Path to the MP3 or Dir: ")
			scanner.Scan()

			filepath = scanner.Text()
		}
		filepath = strings.Trim(filepath, " \"")

		if !OsExists(filepath) {
			fmt.Printf("MP3 or Dir '%s' does not exist.", filepath)
			retcode = 1
			return
		}

		songListPath = append(songListPath, filepath)
	}
	songpaths, _ := GetSongPathList(songListPath)

	if len(songpaths) == 0 {
		fmt.Println("No songs found.")
		retcode = 1
		return
	}

	keysEvents, err := keyboard.GetKeys(3)
	if err != nil {
		fmt.Println(err)
		retcode = 1
		return
	}
	defer keyboard.Close()

	sr := beep.SampleRate(44100)
	speaker.Init(sr, sr.N(time.Second/10))
	defer speaker.Close()

	var queue Queue
	seconds := time.Tick(time.Second)
	next := make(chan bool)
	speaker.Play(&queue)

	songList := make([]*Song, 0)
	bar := progressbar.NewOptions(len(songpaths), progressbarGenericOption...)
	bar.Describe("Reading Files...")
	for _, songpath := range songpaths {
		bar.Describe(fmt.Sprintf("Reading: %s...", filepath.Base(songpath)))
		song, err := NewSong(songpath, sr)
		if err != nil {
			bar.Clear()
			fmt.Printf("Found a problem with '%s', skipped. (%s)\n", filepath.Base(songpath), err)
			continue
		}
		defer song.streamer.Close()

		songList = append(songList, song)
		bar.Add(1)
	}
	bar.Finish()
	bar.Clear()

	done := false
	gonext := false
	fmt.Println("Press [ESC] to exit.")
	var currentVolume float64 = -1
	var volumestep float64 = 0.5
	for i, song := range songList {
		// fmt.Println("Play: ", filepath.Base(song.name))
		speaker.Lock()

		song.volume.Volume = currentVolume
		queue.Add(song.volume, beep.Callback(func() {
			next <- true
		}))

		speaker.Unlock()

		bar := progressbar.NewOptions(int(song.length.Seconds()), progressbarGenericOption...)
		bar.Describe(fmt.Sprintf("[cyan][%v/%v][reset] %v", i+1, len(songList), filepath.Base(song.name)))

	commandloop:
		for {
			select {
			case <-next:
				if !gonext {
					break commandloop
				} else {
					gonext = false
				}
			case event := <-keysEvents:
				// fmt.Printf("You pressed: rune %q, key %X\r\n", event.Rune, event.Key)
				if event.Key == keyboard.KeyEsc || event.Rune == 'q' {
					done = true
					break commandloop
				}
				if event.Key == keyboard.KeyArrowRight || event.Rune == 'l' {
					gonext = true
					break commandloop
				}
				if event.Rune == 'k' || event.Key == keyboard.KeySpace {
					speaker.Lock()
					song.Pause(!song.ctrl.Paused)
					speaker.Unlock()
				}
				if event.Rune == '+' || event.Key == keyboard.KeyArrowUp {
					speaker.Lock()
					song.volume.Volume += volumestep
					currentVolume += volumestep
					speaker.Unlock()
				}
				if event.Rune == '-' || event.Key == keyboard.KeyArrowDown {
					speaker.Lock()
					song.volume.Volume -= volumestep
					currentVolume -= volumestep
					speaker.Unlock()
				}
			case <-seconds:
				if !bar.IsFinished() && !song.ctrl.Paused {
					bar.Set(int(song.GetPosition().Seconds()))
				}
			}
		}
		bar.Set(bar.GetMax())
		bar.Finish()

		song.streamer.Close()

		//fmt.Println("\nDONE", done, ", GONEXT", gonext, ", QLEN", queue.Len())
		if done {
			speaker.Lock()

			queue.Clear()
			queue.Add(beep.Callback(func() {}))
			bar.Clear()

			speaker.Unlock()

			break
		}
	}

	fmt.Println("END")
}
