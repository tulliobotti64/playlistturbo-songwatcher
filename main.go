package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
	"tullio.com/config"
)

func main() {
	config.SetupConfig()
	w := watcher.New()

	// SetMaxEvents to 1 to allow at most 1 event's to be received
	// on the Event channel per watching cycle.
	//
	// If SetMaxEvents is not set, the default is to send all events.
	w.SetMaxEvents(1)

	// Only notify rename and move events.
	w.FilterOps(watcher.Create, watcher.Rename, watcher.Move, watcher.Remove)

	// Only files that match the regular expression during file listings
	// will be watched.
	// r := regexp.MustCompile(`^.*\.(mp3|MP3|flac|FLAC|wav|WAV)$`)
	r := regexp.MustCompile(`^.*\.(mp3|MP3)$`)
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	go func() {
		for {
			select {
			case event := <-w.Event:
				processEvent(event)
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	log.Println("inicio em:", time.Now().UTC())
	if err := w.AddRecursive(config.Config.WatchDir); err != nil {
		log.Fatalln(err)
	}
	log.Println("acabou em:", time.Now().UTC())

	// Trigger 2 events after watcher started.
	go func() {
		w.Wait()
		w.TriggerEvent(watcher.Create, nil)
		// w.TriggerEvent(watcher.Write, nil)
		w.TriggerEvent(watcher.Remove, nil)
		w.TriggerEvent(watcher.Rename, nil)
		w.TriggerEvent(watcher.Move, nil)
	}()

	if err := w.Start(time.Second * time.Duration(config.Config.WatchTimeSec)); err != nil {
		log.Fatalln(err)
	}
}

type importBody struct {
	Path          string `json:"path"`
	Recursive     bool   `json:"recursive"`
	Extension     string `json:"songExtension"`
	GenreFromPath bool   `json:"genreFromPath"`
}
type moveBody struct {
	NewPath   string `json:"newPath"`
	OldPath   string `json:"oldPath"`
	Recursive bool   `json:"recursive"`
	Extension string `json:"songExtension"`
}

func processEvent(event watcher.Event) error {
	// fmt.Println("event:", event)
	var err error
	if event.Op.String() == "CREATE" && len(event.Path) > 1 {
		err = importNewSongs(event)
	}
	if event.Op.String() == "MOVE" && len(event.Path) > 1 {
		if strings.Contains(event.Path, ".Trash") {
			err = removeSongs(event, true)
		} else {
			err = moveSongs(event)
		}
	}
	if event.Op.String() == "REMOVE" && len(event.Path) > 1 {
		err = removeSongs(event, false)
	}

	return err
}

func importNewSongs(event watcher.Event) error {
	paths := strings.Split(event.Path, "/")
	var newpath string
	for x := 1; x < len(paths)-1; x++ {
		piece := "/" + paths[x]
		newpath += piece
	}
	newpath = strings.ReplaceAll(newpath, "'", `\\'`)

	log.Println("new songs detected:", newpath)

	ib := importBody{
		Path:          newpath,
		Recursive:     true,
		Extension:     "mp3",
		GenreFromPath: true,
	}

	body, err := json.Marshal(ib)
	if err != nil {
		log.Println("erro no marshal", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, config.Config.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		log.Println("erro no Post", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("erro no Do", err)
		return err
	}

	if resp.StatusCode != 200 {
		log.Println("erro nao deu 200", err)
		return err
	}
	return nil
}

func moveSongs(event watcher.Event) error {
	if !strings.Contains(event.OldPath, ".mp3") {
		err := errors.New("forbidden to move folders only mp3")
		return err
	}

	var newPath, oldPath string

	if strings.Contains(event.Path, "'") {
		newPath = strings.ReplaceAll(event.Path, "'", `\\'`)
	} else {
		newPath = event.Path
	}

	if strings.Contains(event.OldPath, "'") {
		oldPath = strings.ReplaceAll(event.OldPath, "'", `\\'`)
	} else {
		oldPath = event.OldPath
	}

	log.Println("move songs detected, oldpath:", oldPath)
	log.Println("move songs detected, newpath:", newPath)

	ib := moveBody{
		NewPath:   newPath,
		OldPath:   oldPath,
		Recursive: false,
		Extension: "mp3",
	}

	body, err := json.Marshal(ib)
	if err != nil {
		log.Println("erro no marshal", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPut, config.Config.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		log.Println("erro no Put", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("erro no Do", err)
		return err
	}

	if resp.StatusCode != 200 {
		log.Println("erro nao deu 200", err)
		return err
	}
	return nil
}

func removeSongs(event watcher.Event, oldPath bool) error {

	xPath := ""
	if oldPath {
		xPath = event.OldPath
	} else {
		xPath = event.Path
	}
	if strings.Contains(xPath, "'") {
		xPath = strings.ReplaceAll(xPath, "'", `\\'`)
	}
	log.Println("remove songs detected:", xPath)

	ib := importBody{
		Path:          xPath,
		Recursive:     false,
		Extension:     "mp3",
		GenreFromPath: false,
	}

	body, err := json.Marshal(ib)
	if err != nil {
		log.Println("erro no marshal", err)
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, config.Config.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		log.Println("erro no Delete", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("erro no Do", err)
		return err
	}

	if resp.StatusCode != 200 {
		log.Println("erro nao deu 200", err)
		return err
	}
	return nil
}
