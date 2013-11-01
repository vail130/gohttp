package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

// Application state
type Application struct {
	Name            string
	Version         string
	Commands        []string
	RequestMethods  []string
	Args            []string
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	Mode            string
	HistoryMode     string
	HistoryRecordId int
	HistoryPath     string
	InputFilePath   string
	OutputFilePath  string
	Request         Request
	Response        Response
}

// Single-call entry point
func Start() error {
	home := os.Getenv("HOME")
	historyPath := path.Join(home, ".gohttp/history")

	app := &Application{
		Name:           "gohttp",
		Version:        "0.1.1",
		Commands:       []string{"help", "version", "history"},
		RequestMethods: []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"},
		Args:           os.Args[1:],
		HistoryPath:    historyPath,
	}

	err := app.Run()
	if err != nil {
		return err
	}

	return nil
}

// Application control flow method
func (app *Application) Run() error {
	startTime := time.Now()
	app.StartTime = startTime

	err := app.SetupAppDirs()
	if err != nil {
		return err
	}

	err = app.DetermineMode()
	if err != nil {
		return err
	}

	if app.Mode == "help" {
		app.RunHelp()
	} else if app.Mode == "version" {
		app.RunVersion()
	} else if app.Mode == "history" {
		err := app.RunHistory()
		if err != nil {
			return err
		}
	} else if app.Mode == "http" {
		err := app.RunHttp()
		if err != nil {
			return err
		}
	} else {
		// Default to help
		app.RunHelp()
	}

	return nil
}

// Make sure application dependency directories exist
func (app *Application) SetupAppDirs() error {
	err := os.MkdirAll(app.HistoryPath, 0777)
	if err != nil {
		return errors.New("Failed to create directory " + app.HistoryPath + "\n" + err.Error())
	}
	return nil
}

// Determine desired operation
func (app *Application) DetermineMode() error {
	if len(app.Args) < 1 {
		app.Mode = "help"
	} else {
		for i, j := 0, len(app.Commands); i < j; i++ {
			if strings.ToLower(app.Args[0]) == app.Commands[i] {
				app.Mode = strings.ToLower(app.Args[0])
				break
			}
		}

		if app.Mode == "" {
			app.Mode = "http"
		}
	}
	return nil
}

// Print version to console
func (app *Application) RunVersion() error {
	fmt.Println(app.Name, "version", app.Version)
	return nil
}

// Print help text to console
func (app *Application) RunHelp() error {
	fmt.Println("Usage:")
	fmt.Println("	gohttp COMMAND OPTIONS")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("	help")
	fmt.Println("	version")
	fmt.Println("	history FLAGS")
	fmt.Println("	URL FLAGS")
	fmt.Println("	get URL FLAGS")
	fmt.Println("	head URL FLAGS")
	fmt.Println("	post URL FLAGS")
	fmt.Println("	put URL FLAGS")
	fmt.Println("	patch URL FLAGS")
	fmt.Println("	delete URL FLAGS")
	fmt.Println("")
	fmt.Println("History Flags:")
	fmt.Println("	(-f | --find) GET")
	fmt.Println("	(-i | --insensitive)")
	fmt.Println("	(-l | --limit) 10")
	fmt.Println("	(-s | --skip) 10")
	fmt.Println("")
	fmt.Println("HTTP Flags:")
	fmt.Println("	(-j | --json)")
	fmt.Println("	(-c | --content-type) application/json")
	fmt.Println("	(-a | --accept) application/json")
	fmt.Println("	(-t | --timeout) 0 - 4294967295")
	fmt.Println("	(-i | --input) /path/to/input/file.json")
	fmt.Println("	(-o | --output) /path/to/output/file.json")
	fmt.Println("	(-d | --data) '{\"key\": \"value\"}'")
	fmt.Println("	(-p | --print)")
	fmt.Println("")
	return nil
}

// Determine history mode
func (app *Application) RunHistory() error {
	historyModeMap := map[string]bool{
		"list":   true,
		"detail": true,
		"replay": true,
		"save":   true,
	}

	app.HistoryMode = "list"
	if len(app.Args) > 1 {
		lowerArg := strings.ToLower(app.Args[1])
		if _, present := historyModeMap[lowerArg]; present {
			app.HistoryMode = lowerArg
		}
	}

	if app.HistoryMode == "detail" {
		err := app.RunHistoryDetail()
		if err != nil {
			return err
		}
	} else if app.HistoryMode == "replay" {
		err := app.RunHistoryReplay()
		if err != nil {
			return err
		}
	} else if app.HistoryMode == "save" {
		err := app.RunHistorySave()
		if err != nil {
			return err
		}
	} else {
		// Default to list
		err := app.RunHistoryList()
		if err != nil {
			return err
		}
	}

	return nil
}

// Run HTTP mode flow
func (app *Application) RunHttp() error {
	err := app.CreateRequest()
	if err != nil {
		return err
	}

	err = app.SendRequest()
	if err != nil {
		return err
	}

	err = app.SaveApp()
	if err != nil {
		return err
	}

	return nil
}

// Save app to json file
func (app *Application) SaveApp() error {
	endTime := time.Now()
	duration := endTime.Sub(app.StartTime)
	app.EndTime = endTime
	app.Duration = duration

	fileName := app.getFileName()
	err := app.saveJson(app.HistoryPath, fileName, app)
	if err != nil {
		return err
	}

	return nil
}

//
//	Private functions
//

// Form history filename
func (app *Application) getFileName() string {
	cleanTime := strings.Replace(app.StartTime.String()[:19], ":", "_", -1)
	cleanTime = strings.Replace(cleanTime, " ", "_", -1)
	cleanTime = strings.Replace(cleanTime, "-", "_", -1)
	fileNameSlice := []string{cleanTime, "__", app.Request.Method, "__", cleanUrl(app.Request.URL.String())}
	fileName := strings.Join(fileNameSlice, "")
	if len(fileName) > 196 {
		fileName = fileName[:196]
	}
	return fileName + ".json"
}

// Clean URL for file name
func cleanUrl(url string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]")
	cleanUrl := re.ReplaceAllString(url, "_")
	re = regexp.MustCompile("_+")
	return re.ReplaceAllString(cleanUrl, "_")
}

// Determine if flag is active from command line args
func (app *Application) flagIsActive(flagMap map[string]bool) bool {
	flagIsActive := false
	for i, j := 0, len(app.Args); i < j; i++ {
		if _, present := flagMap[app.Args[i]]; present {
			flagIsActive = true
		}
	}
	return flagIsActive
}

// Get value for command line option
func (app *Application) getOption(optMap map[string]bool, defaultValue string) string {
	optValue := defaultValue
	for i, j := 0, len(app.Args); i < j; i++ {
		if _, present := optMap[app.Args[i]]; present && len(app.Args) > i {
			optValue = app.Args[i+1]
			break
		}
	}
	return optValue
}

// Save object to a file
func (app *Application) saveJson(savePath string, fileName string, v interface{}) error {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return errors.New("Error creating response json: " + err.Error())
	}
	numJsonBytes := len(jsonBytes)

	file, err := os.Create(path.Join(savePath, fileName))
	if err != nil {
		return errors.New("Error creating new " + fileName + " file: " + err.Error())
	}
	defer file.Close()

	numBytesWritten, err := file.Write(jsonBytes)
	if err != nil {
		return errors.New("Error writing json data to file: " + err.Error())
	}

	if numBytesWritten < numJsonBytes {
		return errors.New("Error writing json data to file: Not all data written to file.")
	}

	return nil
}
