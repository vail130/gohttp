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
	"regexp"
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
	RequestPath    string
	ResponsePath   string
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
}

// Response data
type Response struct {
	ContentType   string
	ContentLength uint32
	Body          []byte
	Request       Request
}

// Clean URL for file name
func cleanUrl(url string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]")
	cleanUrl := re.ReplaceAllString(url, "_")
	re = regexp.MustCompile("_+")
	return re.ReplaceAllString(cleanUrl, "_")
}

// Form file name prefix for request
func (req *Request) getFilePrefix() string {
	filePrefix := []string{"request__", req.Method, "__", cleanUrl(req.Url), "__"}
	return strings.Join(filePrefix, "")
}

// Form file name prefix for response
func (resp *Response) getFilePrefix() string {
	filePrefix := []string{"response__", resp.Request.Method, "__", cleanUrl(resp.Request.Url), "__"}
	return strings.Join(filePrefix, "")
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
			defaultValue = app.Args[i+1]
		}
	}
	return optValue
}

// Save object to a file
func (app *Application) save(savePath string, prefix string, v interface{}) error {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return errors.New("Error creating response json: " + err.Error())
	}
	numJsonBytes := len(jsonBytes)

	now := time.Now()
	cleanTime := strings.Replace(now.String()[:19], ":", "_", -1)
	cleanTime = strings.Replace(cleanTime, " ", "_", -1)
	cleanTime = strings.Replace(cleanTime, "-", "_", -1)
	fileName := prefix + cleanTime + ".json"

	file, err := os.Create(path.Join(savePath, fileName))
	if err != nil {
		return errors.New("Error creating new " + prefix + " file: " + err.Error())
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
	err := os.MkdirAll(app.RequestPath, 0777)
	if err != nil {
		return errors.New("Failed to create directory " + app.RequestPath + "\n" + err.Error())
	}

	err = os.MkdirAll(app.ResponsePath, 0777)
	if err != nil {
		return errors.New("Failed to create directory " + app.ResponsePath + "\n" + err.Error())
	}

	return nil
}

// Determine desired operation
func (app *Application) DetermineMode() error {
	if len(app.Args) < 1 {
		return errors.New("Invalid arguments. Try 'gohttp help' for usage details.")
	}

	for i, j := 0, len(app.Commands); i < j; i++ {
		if app.Args[0] == app.Commands[i] {
			app.Mode = app.Args[0]
			break
		}
	}

	if app.Mode == "" {
		app.Mode = "http"
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
	fmt.Println("	version")
	fmt.Println("	help")
	fmt.Println("	history")
	fmt.Println("	URL FLAGS")
	fmt.Println("	get URL FLAGS")
	fmt.Println("	head URL FLAGS")
	fmt.Println("	post URL FLAGS")
	fmt.Println("	put URL FLAGS")
	fmt.Println("	patch URL FLAGS")
	fmt.Println("	delete URL FLAGS")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("	-j | --json")
	fmt.Println("	(-c | --content-type) application/json")
	fmt.Println("	(-t | --timeout) 0 - 4294967295")
	fmt.Println("	(-i | --input) /path/to/input/file.json")
	fmt.Println("	(-o | --output) /path/to/output/file.json")
	fmt.Println("")
	return nil
}

// Show reverse chronological requests/responses
func (app *Application) ShowHistory() error {

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
	inputFileSize := int64(0)
	outputFilePath := app.getOption(outputFlagMap, "")
	jsonContentType := app.flagIsActive(jsonFlagMap)
	contentType := app.getOption(contentTypeFlagMap, "application/json")

	if inputFilePath != "" {
		if fileInfo, err := os.Stat(inputFilePath); os.IsNotExist(err) {
			inputFilePath = ""
		} else {
			inputFileSize = fileInfo.Size()
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
		Timeout:       60,
		ContentType:   requestContentType,
		ContentLength: inputFileSize,
	}

	err = app.save(app.RequestPath, app.Request.getFilePrefix(), app.Request)
	if err != nil {
		return err
	}

	return nil
}

// Send HTTP request
func (app *Application) SendRequest() error {
	fmt.Println("Sending request...")

	emptyBytes := make([]byte, 0)
	requestData := bytes.NewReader(emptyBytes)

	if app.InputFilePath != "" {
		body, err := os.Open(app.InputFilePath)
		if err != nil {
			return errors.New("Error opening file " + app.InputFilePath + "\n" + err.Error())
		}
		defer body.Close()

		data := make([]byte, app.Request.ContentLength)
		_, err = body.Read(data)
		if err != nil {
			return errors.New("Error reading input file: " + err.Error())
		}
		requestData = bytes.NewReader(data)
	}

	req, err := http.NewRequest(app.Request.Method, app.Request.Url, requestData)
	if err != nil {
		return errors.New("Error making new request object: " + err.Error())
	}
	if app.Request.ContentType != "" {
		req.Header.Add("Content-Type", app.Request.ContentType)
	}

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

	app.Response = Response{
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: uint32(len(responseData)),
		Body:          responseData,
		Request:       app.Request,
	}

	err = app.save(app.ResponsePath, app.Response.getFilePrefix(), app.Response)
	if err != nil {
		return err
	}

	return nil
}

// Save HTTP response body to output file, if specified
func (app *Application) SaveResponse() error {
	if app.OutputFilePath != "" {
		fmt.Println("Saving response data to output file...")
		outputFile := new(os.File)
		if _, err := os.Stat(app.OutputFilePath); os.IsNotExist(err) {
			outputFile, err = os.Create(app.OutputFilePath)
			if err != nil {
				return errors.New("Error creating new output file: " + err.Error())
			}
		} else {
			outputFile, err = os.Open(app.OutputFilePath)
			if err != nil {
				return errors.New("Error opening output file " + app.OutputFilePath + "\n" + err.Error())
			}
		}
		defer outputFile.Close()

		_, err := outputFile.Write(app.Response.Body)
		if err != nil {
			return errors.New("Error writing to output file: " + err.Error())
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

		err = app.SaveResponse()
		if err != nil {
			return err
		}
	} else {
		return errors.New("Invalid application mode. Use gohttp help for usage.")
	}

	return nil
}

// Publicly exposed package entry point
func Start() error {
	home := os.Getenv("HOME")
	requestPath := path.Join(home, ".gohttp/requests")
	responsePath := path.Join(home, ".gohttp/responses")

	app := &Application{
		Name:           "gohttp",
		Version:        "0.1.0",
		Commands:       []string{"version", "help", "history"},
		RequestMethods: []string{"head", "get", "post", "put", "patch", "delete"},
		Args:           os.Args[1:],
		RequestPath:    requestPath,
		ResponsePath:   responsePath,
	}

	err := app.Run()
	if err != nil {
		return err
	}

	return nil
}
