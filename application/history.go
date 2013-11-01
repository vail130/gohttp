package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

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
	skip, limit, find, caseFlag := app.getHistoryListOptions()
	items, itemIndexes, numTotal, numSkipped, err := app.getHistoryRecords(skip, limit, find, caseFlag)
	if err != nil {
		return err
	}

	if numTotal == 0 {
		fmt.Println("Nothing in history.")
	} else {
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

//
//	Private functions
//

func (app *Application) getHistoryListOptions() (int, int, string, bool) {
	var err error

	skipOptMap := map[string]bool{
		"-s":     true,
		"--skip": true,
	}
	skipOpt := app.getOption(skipOptMap, "")
	skip := 0
	if skipOpt != "" {
		skip, err = strconv.Atoi(skipOpt)
		if err != nil || skip < 0 {
			skip = 0
		}
	}

	limitOptMap := map[string]bool{
		"-l":      true,
		"--limit": true,
	}
	limitOpt := app.getOption(limitOptMap, "")
	limit := 10
	if limitOpt != "" {
		limit, err = strconv.Atoi(limitOpt)
		if err != nil || limit < 1 {
			limit = 10
		}
	}

	findOptMap := map[string]bool{
		"-f":     true,
		"--find": true,
	}
	find := app.getOption(findOptMap, "")

	caseFlagMap := map[string]bool{
		"-i":            true,
		"--insensitive": true,
	}
	caseFlag := app.flagIsActive(caseFlagMap)

	return skip, limit, find, caseFlag
}

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
