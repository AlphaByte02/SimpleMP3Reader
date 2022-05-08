package main

import (
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
)

type Song struct {
	name string

	length    time.Duration
	streamer  beep.StreamSeekCloser
	resampled *beep.Resampler
	ctrl      *beep.Ctrl
	volume    *effects.Volume
	format    *beep.Format
}

func NewSong(songName string, sampleRate beep.SampleRate) (*Song, error) {
	f, err := os.Open(songName)
	if err != nil {
		return nil, err
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return nil, err
	}

	length := format.SampleRate.D(streamer.Len()).Round(time.Second)
	ctrl := &beep.Ctrl{Streamer: streamer, Paused: false}
	resampled := beep.Resample(4, format.SampleRate, sampleRate, ctrl)
	volume := &effects.Volume{Streamer: resampled, Base: 2}
	s := &Song{songName, length, streamer, resampled, ctrl, volume, &format}

	return s, nil
}

func (s *Song) GetPosition() time.Duration {
	return s.format.SampleRate.D(s.streamer.Position())
}

func (s *Song) Pause(value bool) {
	s.ctrl.Paused = value
}

func (s *Song) Volume(value float64) {
	s.volume.Volume = value
}
