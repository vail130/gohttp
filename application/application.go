package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
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
		Version:        "0.1.0",
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
	fmt.Println("")
	return nil
}

// Determine history mode
func (app *Application) RunHistory() error {
	historyModeMap := map[string]bool{
		"list":   true,
		"detail": true,
		"replay": true,
	}

	historyMode := "list"
	if len(app.Args) > 1 {
		lowerArg := strings.ToLower(app.Args[1])
		if _, present := historyModeMap[lowerArg]; present {
			historyMode = lowerArg
		}
	}

	app.HistoryMode = historyMode

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
	} else {
		// Default to list
		err := app.RunHistoryList()
		if err != nil {
			return err
		}
	}

	return nil
}

// Show details of history request/response
func (app *Application) RunHistoryDetail() error {
	historyApp, err := app.loadAppFromHistory()
	if err != nil {
		return err
	}

	fmt.Println("Name:", historyApp.Name)
	fmt.Println("Version:", historyApp.Version)
	fmt.Println("Args:", historyApp.Args)
	fmt.Println("Mode:", historyApp.Mode)
	fmt.Println("Start Time:", historyApp.StartTime)
	fmt.Println("End Time:", historyApp.EndTime)
	fmt.Println("Duration:", historyApp.Duration)
	fmt.Println("InputFilePath:", historyApp.InputFilePath)
	fmt.Println("OutputFilePath:", historyApp.OutputFilePath)

	fmt.Println("Request Method:", historyApp.Request.Method)
	fmt.Println("Request URL:", historyApp.Request.URL)
	fmt.Println("Request Timeout:", historyApp.Request.Timeout)
	fmt.Println("Request Content Type:", historyApp.Request.ContentType)
	fmt.Println("Request Accept:", historyApp.Request.Accept)

	fmt.Println("Response Content Type:", historyApp.Response.ContentType)
	fmt.Println("Response Content Length:", historyApp.Response.ContentLength)

	return nil
}

// Replay a request from history
func (app *Application) RunHistoryReplay() error {
	historyApp, err := app.loadAppFromHistory()
	if err != nil {
		return err
	}

	app.Request = historyApp.Request

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

// Show reverse chronological requests/responses
func (app *Application) RunHistoryList() error {
	findOptMap := map[string]bool{
		"-f":     true,
		"--find": true,
	}
	caseFlagMap := map[string]bool{
		"-i":            true,
		"--insensitive": true,
	}
	limitOptMap := map[string]bool{
		"-l":      true,
		"--limit": true,
	}
	skipOptMap := map[string]bool{
		"-s":     true,
		"--skip": true,
	}

	findOpt := app.getOption(findOptMap, "")
	caseFlag := app.flagIsActive(caseFlagMap)
	limitOpt := app.getOption(limitOptMap, "")
	skipOpt := app.getOption(skipOptMap, "")
	limit := 10
	var err error
	if limitOpt != "" {
		limit, err = strconv.Atoi(limitOpt)
		if err != nil || limit < 1 {
			limit = 10
		}
	}
	skip := 0
	if skipOpt != "" {
		skip, err = strconv.Atoi(skipOpt)
		if err != nil || skip < 0 {
			skip = 0
		}
	}

	items, itemIndexes, numTotal, numSkipped, err := app.getHistoryRecords(skip, limit, findOpt, caseFlag)
	if numTotal == 0 {
		fmt.Println("Nothing in history.")
	} else {
		if err != nil {
			return err
		}

		if len(items) == 0 {
			fmt.Println("No results matching criteria.")
		} else {
			fmt.Println("Displaying", numSkipped+1, "to", numSkipped+1+len(items), "of", numTotal, "-", "Use skip and limit flags to page.")
			fmt.Println("")
			for i, j := 0, len(items); i < j; i++ {
				fmt.Println(strconv.Itoa(itemIndexes[i]) + ". " + items[i].Name())
			}
		}
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

func (app *Application) getHistoryRecords(skip int, limit int, find string, caseInsensitive bool) ([]os.FileInfo, []int, int, int, error) {
	itemIndex := 0
	numTotal := 0
	numSkipped := 0
	items := make([]os.FileInfo, 0, limit)
	itemIndexes := make([]int, 0, limit)

	fileInfos, err := ioutil.ReadDir(app.HistoryPath)
	if err != nil {
		return items, itemIndexes, numSkipped, numTotal, err
	}

	numTotal = len(fileInfos)
	for i, j := len(fileInfos)-1, 0; i >= j && (limit < 1 || len(items) < limit); i-- {
		fileName := fileInfos[i].Name()
		if len(fileName) > 0 && string(fileName[0]) != "." {
			flagAndLowerExists := caseInsensitive && strings.Index(strings.ToLower(fileName), strings.ToLower(find)) > -1
			if numSkipped >= skip && (find == "" || flagAndLowerExists || strings.Index(fileName, find) > -1) {
				items = append(items, fileInfos[i])
				label := itemIndex + 1
				itemIndexes = append(itemIndexes, label)
			} else {
				numSkipped++
			}
			// Keep numbers consistent for history items, regardless if filtering
			itemIndex++
		}
	}

	return items, itemIndexes, numTotal, numSkipped, nil
}

// Load an app object from history file
func (app *Application) loadAppFromHistory() (Application, error) {
	historyApp := Application{}

	if len(app.Args) < 3 {
		return historyApp, errors.New("Missing history record index.")
	}

	historyIndex, err := strconv.Atoi(app.Args[2])
	if err != nil {
		return historyApp, err
	}
	app.HistoryRecordId = historyIndex

	skip := 0
	if historyIndex > 1 {
		skip = historyIndex - 1
	}
	limit := 1

	items, itemIndexes, _, _, err := app.getHistoryRecords(skip, limit, "", true)
	if err != nil {
		return historyApp, err
	} else if len(items) != 1 || len(itemIndexes) != 1 {
		return historyApp, errors.New("No history records found.")
	} else if historyIndex != itemIndexes[0] {
		return historyApp, errors.New("Invalid history record index: " + app.Args[2])
	}
	fileName := items[0].Name()
	fileSize := items[0].Size()

	file, err := os.Open(path.Join(app.HistoryPath, fileName))
	if err != nil {
		return historyApp, errors.New("Error opening history file " + fileName + "\n" + err.Error())
	}
	defer file.Close()

	fileData := make([]byte, fileSize)
	numBytesRead, err := file.Read(fileData)
	if err != nil {
		return historyApp, errors.New("Error reading history file: " + err.Error())
	}

	if numBytesRead < int(fileSize) {
		return historyApp, errors.New("Error reading history file: Read " +
			strconv.Itoa(numBytesRead) + " out of " + strconv.Itoa(int(fileSize)) + "bytes.")
	}

	err = json.Unmarshal(fileData, &historyApp)
	if err != nil {
		return historyApp, errors.New("Error unmarshalling json: " + err.Error())
	}

	return historyApp, nil
}

// Clean URL for file name
func cleanUrl(url string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]")
	cleanUrl := re.ReplaceAllString(url, "_")
	re = regexp.MustCompile("_+")
	return re.ReplaceAllString(cleanUrl, "_")
}

// Form history filename
func (app *Application) getFileName() string {
	cleanTime := strings.Replace(app.StartTime.String()[:19], ":", "_", -1)
	cleanTime = strings.Replace(cleanTime, " ", "_", -1)
	cleanTime = strings.Replace(cleanTime, "-", "_", -1)
	fileNameSlice := []string{cleanTime, "__", app.Request.Method, "__", cleanUrl(app.Request.URL.String()), ".json"}
	fileName := strings.Join(fileNameSlice, "")
	if len(fileName) > 200 {
		fileName = fileName[:200]
	}
	return fileName
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
