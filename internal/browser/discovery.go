package browser

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"historylibhelper/internal/model"
)

type chromiumLocation struct{ name, path string }

func Discover() ([]model.Profile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var profiles []model.Profile
	for _, location := range chromiumLocations(home) {
		profiles = append(profiles, discoverChromium(location)...)
	}
	profiles = append(profiles, discoverFirefox(home)...)
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Browser == profiles[j].Browser {
			return profiles[i].Name < profiles[j].Name
		}
		return profiles[i].Browser < profiles[j].Browser
	})
	return profiles, nil
}

func ManualProfile(database, engine, browserName string) (model.Profile, error) {
	abs, err := filepath.Abs(database)
	if err != nil {
		return model.Profile{}, err
	}
	if _, err := os.Stat(abs); err != nil {
		return model.Profile{}, err
	}
	engine = strings.ToLower(engine)
	if engine != "chromium" && engine != "firefox" {
		return model.Profile{}, fmt.Errorf("engine must be chromium or firefox / 引擎必须是 chromium 或 firefox")
	}
	if browserName == "" {
		browserName = "Chromium"
		if engine == "firefox" {
			browserName = "Firefox"
		}
	}
	return makeProfile(browserName, "Manual", abs, engine), nil
}

func discoverChromium(location chromiumLocation) []model.Profile {
	entries, err := os.ReadDir(location.path)
	if err != nil {
		return nil
	}
	var result []model.Profile
	if exists(filepath.Join(location.path, "History")) {
		result = append(result, makeProfile(location.name, "Default", filepath.Join(location.path, "History"), "chromium"))
	}
	for _, entry := range entries {
		if !entry.IsDir() || (entry.Name() != "Default" && !strings.HasPrefix(entry.Name(), "Profile ")) {
			continue
		}
		db := filepath.Join(location.path, entry.Name(), "History")
		if exists(db) {
			result = append(result, makeProfile(location.name, entry.Name(), db, "chromium"))
		}
	}
	return result
}

func discoverFirefox(home string) []model.Profile {
	var root string
	switch runtime.GOOS {
	case "darwin":
		root = filepath.Join(home, "Library/Application Support/Firefox/Profiles")
	case "windows":
		root = filepath.Join(os.Getenv("APPDATA"), "Mozilla/Firefox/Profiles")
	default:
		root = filepath.Join(home, ".mozilla/firefox")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var result []model.Profile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		db := filepath.Join(root, entry.Name(), "places.sqlite")
		if exists(db) {
			result = append(result, makeProfile("Firefox", entry.Name(), db, "firefox"))
		}
	}
	return result
}

func chromiumLocations(home string) []chromiumLocation {
	if runtime.GOOS == "windows" {
		local, roaming := os.Getenv("LOCALAPPDATA"), os.Getenv("APPDATA")
		return []chromiumLocation{{"Google Chrome", filepath.Join(local, "Google/Chrome/User Data")}, {"Microsoft Edge", filepath.Join(local, "Microsoft/Edge/User Data")}, {"Brave", filepath.Join(local, "BraveSoftware/Brave-Browser/User Data")}, {"Vivaldi", filepath.Join(local, "Vivaldi/User Data")}, {"Opera", filepath.Join(roaming, "Opera Software/Opera Stable")}}
	}
	if runtime.GOOS == "darwin" {
		base := filepath.Join(home, "Library/Application Support")
		return []chromiumLocation{{"Google Chrome", filepath.Join(base, "Google/Chrome")}, {"Microsoft Edge", filepath.Join(base, "Microsoft Edge")}, {"Brave", filepath.Join(base, "BraveSoftware/Brave-Browser")}, {"Vivaldi", filepath.Join(base, "Vivaldi")}, {"Opera", filepath.Join(base, "com.operasoftware.Opera")}, {"Chromium", filepath.Join(base, "Chromium")}}
	}
	base := filepath.Join(home, ".config")
	return []chromiumLocation{{"Google Chrome", filepath.Join(base, "google-chrome")}, {"Microsoft Edge", filepath.Join(base, "microsoft-edge")}, {"Brave", filepath.Join(base, "BraveSoftware/Brave-Browser")}, {"Vivaldi", filepath.Join(base, "vivaldi")}, {"Opera", filepath.Join(base, "opera")}, {"Chromium", filepath.Join(base, "chromium")}}
}

func makeProfile(browserName, profileName, database, engine string) model.Profile {
	hash := sha256.Sum256([]byte(database))
	return model.Profile{ID: fmt.Sprintf("%x", hash[:8]), Browser: browserName, Name: profileName, Database: database, Engine: engine}
}
func exists(path string) bool { info, err := os.Stat(path); return err == nil && !info.IsDir() }
