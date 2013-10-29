package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Application state
type Application struct {
	Name           string
	Version        string
	Commands       []string
	RequestMethods []string
	Args           []string
	Mode           string
	HistoryPath    string
	InputFilePath  string
	OutputFilePath string
	Request        Request
	Response       Response
}

// Data about the request to send
type Request struct {
	Method        string
	Url           string
	Timeout       uint32
	ContentType   string
	ContentLength int64
	RawRequest    *http.Request
}

// Response data
type Response struct {
	ContentType   string
	ContentLength uint32
	Body          []byte
	RawResponse   *http.Response
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
	now := time.Now()
	cleanTime := strings.Replace(now.String()[:19], ":", "_", -1)
	cleanTime = strings.Replace(cleanTime, " ", "_", -1)
	cleanTime = strings.Replace(cleanTime, "-", "_", -1)
	fileName := []string{cleanTime, "__", app.Request.Method, "__", cleanUrl(app.Request.Url), ".json"}
	return strings.Join(fileName, "")
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
			if app.Args[0] == app.Commands[i] {
				app.Mode = app.Args[0]
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
func (app *Application) ShowVersion() error {
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
	fmt.Println("")
	fmt.Println("HTTP Request Flags:")
	fmt.Println("	(-j | --json)")
	fmt.Println("	(-c | --content-type) application/json")
	fmt.Println("	(-t | --timeout) 0 - 4294967295")
	fmt.Println("	(-i | --input) /path/to/input/file.json")
	fmt.Println("	(-o | --output) /path/to/output/file.json")
	fmt.Println("")
	return nil
}

// Show reverse chronological requests/responses
func (app *Application) ShowHistory() error {
	fileInfos, err := ioutil.ReadDir(app.HistoryPath)
	if err != nil {
		return errors.New("Failed to read history directory: " + err.Error())
	}

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

	findOpt := app.getOption(findOptMap, "")
	caseFlag := app.flagIsActive(caseFlagMap)
	limitOpt := app.getOption(limitOptMap, "")
	limit := 0
	if limitOpt != "" {
		limit, err = strconv.Atoi(limitOpt)
		if err != nil {
			limit = 0
		}
	}

	if len(fileInfos) == 0 {
		fmt.Println("Nothing in history.")
	} else {
		itemIndex := 0
		numPrinted := 0
		for i, j := len(fileInfos)-1, 0; i >= j && (limit < 1 || numPrinted < limit); i-- {
			fileName := fileInfos[i].Name()
			if len(fileName) > 0 && string(fileName[0]) != "." {
				flagAndLowerExists := caseFlag && strings.Index(strings.ToLower(fileName), strings.ToLower(findOpt)) > -1
				if findOpt == "" || flagAndLowerExists || strings.Index(fileName, findOpt) > -1 {
					label := strconv.Itoa(itemIndex + 1)
					fmt.Println(label+".", fileName)
					numPrinted++
				}
				// Keep numbers consistent for history items, regardless if filtering
				itemIndex++
			}
		}

		if numPrinted == 0 {
			fmt.Println("No results matching criteria.")
		}
	}

	return nil
}

// Parse command line arguments
func (app *Application) CreateRequest() error {
	fmt.Println("Parsing arguments...")

	inputFlagMap := map[string]bool{
		"-i":      true,
		"--input": true,
	}
	outputFlagMap := map[string]bool{
		"-o":       true,
		"--output": true,
	}
	jsonFlagMap := map[string]bool{
		"-j":     true,
		"--json": true,
	}
	contentTypeFlagMap := map[string]bool{
		"-c":             true,
		"--content-type": true,
	}
	timeoutFlagMap := map[string]bool{
		"-t":        true,
		"--timeout": true,
	}

	requestMethod := app.RequestMethods[0]
	requestMethodProvided := false
	for i, j := 0, len(app.RequestMethods); i < j; i++ {
		if app.RequestMethods[i] == app.Args[0] {
			requestMethod = app.Args[0]
			requestMethodProvided = true
			break
		}
	}

	urlIndex := 0
	if requestMethodProvided {
		urlIndex = 1
	}
	if len(app.Args) < urlIndex+1 {
		return errors.New("Invalid arguments. Try 'gohttp help' for usage details.")
	}
	_, err := url.Parse(app.Args[urlIndex])
	if err != nil {
		return errors.New("Error parsing URL: " + err.Error())
	}

	requestUrl := app.Args[urlIndex]
	inputFilePath := app.getOption(inputFlagMap, "")
	contentLength := int64(0)
	requestData := make([]byte, 0)
	outputFilePath := app.getOption(outputFlagMap, "")
	jsonContentType := app.flagIsActive(jsonFlagMap)
	contentType := app.getOption(contentTypeFlagMap, "application/json")
	timeoutOpt := app.getOption(timeoutFlagMap, "0")
	timeout, err := strconv.Atoi(timeoutOpt)
	if err != nil || timeout < 1 {
		timeout = 60
	}

	if inputFilePath != "" {
		if fileInfo, err := os.Stat(inputFilePath); os.IsNotExist(err) {
			inputFilePath = ""
		} else {
			contentLength = fileInfo.Size()

			body, err := os.Open(inputFilePath)
			if err != nil {
				return errors.New("Error opening file " + inputFilePath + "\n" + err.Error())
			}
			defer body.Close()

			requestData = make([]byte, contentLength)
			numBytesRead, err = body.Read(data)
			if err != nil {
				return errors.New("Error reading input file: " + err.Error())
			}

			if numBytesRead < contentLength {
				return errors.New("Error reading input file: Read " +
					strconv.Itoa(numBytesRead) + " out of " + strconv.Itoa(contentLength) + "bytes.")
			}
		}
	}

	requestContentType := ""
	if jsonContentType {
		requestContentType = "application/json"
	}
	if contentType != "" {
		requestContentType = contentType
	}

	app.InputFilePath = inputFilePath
	app.OutputFilePath = outputFilePath

	app.Request = Request{
		Method:        strings.ToUpper(requestMethod),
		Url:           requestUrl,
		Timeout:       timeout,
		ContentType:   requestContentType,
		ContentLength: contentLength,
		Body:          requestData,
	}

	return nil
}

// Send HTTP request
func (app *Application) SendRequest() error {
	fmt.Println("Sending request...")

	requestData := bytes.NewReader(app.Request.Body)
	req, err := http.NewRequest(app.Request.Method, app.Request.Url, requestData)
	if err != nil {
		return errors.New("Error making new request object: " + err.Error())
	}
	if app.Request.ContentType != "" {
		req.Header.Add("Content-Type", app.Request.ContentType)
	}

	app.Request.RawRequest = req

	transport := &http.Transport{
		ResponseHeaderTimeout: time.Duration(app.Request.Timeout) * time.Second,
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return errors.New("Error sending request: " + err.Error())
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("Error reading response body: " + err.Error())
	}

	numResponseBytes := len(responseData)
	app.Response = Response{
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: uint32(numResponseBytes),
		Body:          responseData,
		RawResponse:   resp,
	}

	fileName := app.getFileName()
	err = app.saveJson(app.HistoryPath, fileName, app)
	if err != nil {
		return err
	}

	if app.OutputFilePath != "" {
		dirName := filepath.Dir(app.OutputFilePath)

		err := os.MkdirAll(dirName, 0777)
		if err != nil {
			return errors.New("Failed to create directory " + dirName + "\n" + err.Error())
		}

		fileName := filepath.Base(app.OutputFilePath)
		file, err := os.Create(path.Join(dirName, fileName))
		if err != nil {
			return errors.New("Error creating new " + fileName + " file: " + err.Error())
		}
		defer file.Close()

		numBytesWritten, err := file.Write(responseData)
		if err != nil {
			return errors.New("Error writing json data to file: " + err.Error())
		}

		if numBytesWritten < numResponseBytes {
			return errors.New("Error writing data to output file: Not all data written to file.")
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// Application control flow method
func (app *Application) Run() error {
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
		app.ShowVersion()
	} else if app.Mode == "history" {
		err := app.ShowHistory()
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
	} else {
		// Default to help
		app.RunHelp()
	}

	return nil
}

// Publicly exposed package entry point
func Start() error {
	home := os.Getenv("HOME")
	historyPath := path.Join(home, ".gohttp/history")

	app := &Application{
		Name:           "gohttp",
		Version:        "0.1.0",
		Commands:       []string{"help", "version", "history"},
		RequestMethods: []string{"get", "head", "post", "put", "patch", "delete"},
		Args:           os.Args[1:],
		HistoryPath:    historyPath,
	}

	err := app.Run()
	if err != nil {
		return err
	}

	return nil
}
