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
	"strings"
	"time"
)

// Application state
type Application struct {
	Name           string
	Version        string
	Args           []string
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

/*
 * Private Functions
 */
func (app *Application) flagIsActive(flagMap map[string]bool) bool {
	flagIsActive := false
	for i, j := 0, len(app.Args); i < j; i++ {
		if _, present := flagMap[app.Args[i]]; present {
			flagIsActive = true
		}
	}
	return flagIsActive
}

func (app *Application) getOption(optMap map[string]bool, defaultValue string) string {
	optValue := defaultValue
	for i, j := 0, len(app.Args); i < j; i++ {
		if _, present := optMap[app.Args[i]]; present && len(app.Args) > i {
			defaultValue = app.Args[i+1]
		}
	}
	return optValue
}

func (app *Application) save(v interface{}, savePath string, name string) error {

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return errors.New("Error creating response json: " + err.Error())
	}
	numJsonBytes := len(jsonBytes)

	now := time.Now()
	cleanTime := strings.Replace(now.String()[:19], ":", "_", -1)
	cleanTime = strings.Replace(cleanTime, " ", "_", -1)
	cleanTime = strings.Replace(cleanTime, "-", "_", -1)
	fileName := name + "__" + cleanTime + ".json"

	file, err := os.Create(path.Join(savePath, fileName))
	if err != nil {
		return errors.New("Error creating new " + name + " file: " + err.Error())
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

/*
 * Public Functions
 */
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

func (app *Application) ParseArgs() error {
	fmt.Println("Parsing arguments...")

	if len(app.Args) < 1 {
		return errors.New("No arguments. Try 'gohttp --help' for usage details.")
	}

	helpFlagMap := map[string]bool{
		"-h":     true,
		"--help": true,
	}
	versionFlagMap := map[string]bool{
		"-v":        true,
		"--version": true,
	}
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

	if app.flagIsActive(helpFlagMap) {
		fmt.Println("> gohttp (get | head | post | put | patch | delete) URL [")
		fmt.Println("	[")
		fmt.Println("		[(-j | --json) | ((-c | --content-type) application/json)]")
		fmt.Println("	]")
		fmt.Println("	[(-t | --timeout) 0 - 4294967295]")
		fmt.Println("	[(-i | --input) /path/to/input/file.json]")
		fmt.Println("	[(-o | --output) /path/to/output/file.json]")
		fmt.Println("]")
		os.Exit(0)
	}

	if app.flagIsActive(versionFlagMap) {
		fmt.Println(app.Name, "version", app.Version)
		os.Exit(0)
	}

	requestMethods := make([]string, 0, 5)
	requestMethods = append(requestMethods, "get", "post", "put", "patch", "delete")
	requestMethod := requestMethods[0]
	requestMethodProvided := false
	for i, j := 0, len(requestMethods); i < j; i++ {
		if requestMethods[i] == app.Args[0] {
			requestMethod = app.Args[0]
			requestMethodProvided = true
		}
	}

	urlIndex := 0
	if requestMethodProvided {
		urlIndex = 1
	}
	if len(app.Args) < urlIndex+1 {
		return errors.New("Invalid arguments. Try 'gohttp --help' for usage details.")
	}
	_, err := url.Parse(app.Args[urlIndex])
	if err != nil {
		return errors.New("Error parsing URL: " + err.Error())
	}

	requestUrl := app.Args[urlIndex]
	inputFilePath := app.getOption(inputFlagMap, "")
	inputFileSize := int64(0)
	outputFilePath := app.getOption(outputFlagMap, app.ResponsePath)
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

	err = app.save(app.Request, app.RequestPath, "request")
	if err != nil {
		return err
	}

	return nil
}

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

	err = app.save(app.Response, app.ResponsePath, "response")
	if err != nil {
		return err
	}

	return nil
}

func (app *Application) SaveResponse() error {
	fmt.Println("Saving response...")

	if app.OutputFilePath != "" {
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
