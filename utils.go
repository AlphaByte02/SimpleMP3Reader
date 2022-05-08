package main

import (
	"os"
	"path/filepath"
)

func OsExists(path string) bool {
	_, err := os.Stat(path)
	return err != os.ErrNotExist
}

func IsDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), nil
}

func GetSongPathList(paths []string) ([]string, error) {
	songs := make([]string, 0)

	for _, path := range paths {
		isDir, err := IsDir(path)
		if err != nil {
			return songs, err
		}

		if isDir {
			dir, err := os.Open(path)
			if err != nil {
				return songs, err
			}

			files, _ := dir.Readdirnames(0)
			for _, fileName := range files {
				songpath := filepath.Join(path, fileName)
				isDir, errdir := IsDir(songpath)
				if errdir != nil || isDir || filepath.Ext(songpath) != ".mp3" {
					continue
				}

				songs = append(songs, songpath)
			}
		} else if filepath.Ext(path) == ".mp3" {
			songs = append(songs, path)
		}
	}

	return songs, nil
}
