package apps

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gaetschwartz/devcleaner-go/internal/config"
	"github.com/gaetschwartz/devcleaner-go/internal/utils/log"
)

type Manifest struct {
	Apps        []App     `json:"apps"`
	LastUpdated time.Time `json:"last_updated"`
	Version     int       `json:"version"`
}

type ManifestWithTime struct {
	Manifest Manifest
	ModTime  time.Time
}

func GetLocalManifest() (*ManifestWithTime, error) {
	manifestPath := config.GetLocalManifestPath()
	// check if file exists
	stat, err := os.Stat(manifestPath)
	if err != nil {
		return nil, err
	}
	// read file
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}
	return &ManifestWithTime{Manifest: manifest, ModTime: stat.ModTime()}, nil
}

func FetchManifestFromRemote() (*Manifest, error) {
	manifest_url := config.Config.ManifestURL
	resp, err := http.Get(manifest_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}
func writeToLocalManifest(manifest *Manifest) error {
	filepath := config.GetLocalManifestPath()
	_, err := os.Stat(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// create parent directories
			os.MkdirAll(path.Dir(filepath), 0755)
		} else {
			return err
		}
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0644)
}

func GetManifest(l *log.Logger) (*Manifest, error) {
	if localManifest, err := GetLocalManifest(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			l.Debug("No local manifest found, fetching remote manifest")
		} else {
			return nil, err
		}
	} else if localManifest != nil {
		l.Debug("Found local manifest at %s", config.GetLocalManifestPath())
		// check if its not too old
		if time.Since(localManifest.ModTime) < config.Config.ManifestTTL {
			l.Debug("Local manifest is not too old")
			return &localManifest.Manifest, nil
		}

		l.Debug("Local manifest is too old, fetching remote manifest")
	}
	remoteManifest, err := FetchManifestFromRemote()
	if err != nil {
		return nil, err
	}
	if err := writeToLocalManifest(remoteManifest); err != nil {
		return nil, err
	}
	return remoteManifest, nil
}
