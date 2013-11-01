package application

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Data about the request to send
type Request struct {
	Method        string
	URL           *url.URL
	Timeout       int
	ContentType   string
	Accept        string
	ContentLength int
	Body          []byte
}

// Response data
type Response struct {
	ContentType   string
	ContentLength int
	Body          []byte
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
	contentTypeOptMap := map[string]bool{
		"-c":             true,
		"--content-type": true,
	}
	acceptOptMap := map[string]bool{
		"-a":       true,
		"--accept": true,
	}
	timeoutOptMap := map[string]bool{
		"-t":        true,
		"--timeout": true,
	}
	dataOptMap := map[string]bool{
		"-d":     true,
		"--data": true,
	}

	requestMethod := app.RequestMethods[0]
	requestMethodProvided := false
	for i, j := 0, len(app.RequestMethods); i < j; i++ {
		if app.RequestMethods[i] == strings.ToUpper(app.Args[0]) {
			requestMethod = strings.ToUpper(app.Args[0])
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
	requestUrl, err := url.Parse(app.Args[urlIndex])
	if err != nil {
		return errors.New("Error parsing URL: " + err.Error())
	}
	// URL encode the query string
	query := requestUrl.Query()
	requestUrl.RawQuery = query.Encode()

	inputFilePath := app.getOption(inputFlagMap, "")
	outputFilePath := app.getOption(outputFlagMap, "")
	jsonContentType := app.flagIsActive(jsonFlagMap)
	contentType := app.getOption(contentTypeOptMap, "")
	acceptOpt := app.getOption(acceptOptMap, "")
	dataOpt := app.getOption(dataOptMap, "")
	timeoutOpt := app.getOption(timeoutOptMap, "0")
	timeout, err := strconv.Atoi(timeoutOpt)
	if err != nil || timeout < 1 {
		timeout = 60
	}

	contentLength := 0
	requestData := make([]byte, 0)
	if requestMethod == "POST" || requestMethod == "PATCH" || requestMethod == "PUT" {
		if dataOpt != "" {
			contentLength = len(dataOpt)
			requestData = make([]byte, contentLength)
			reader := strings.NewReader(dataOpt)
			numBytesRead, err := reader.Read(requestData)
			if err != nil {
				return errors.New("Error reading input data: " + err.Error())
			}

			if numBytesRead < contentLength {
				return errors.New("Error reading input data: Read " +
					strconv.Itoa(numBytesRead) + " out of " + strconv.Itoa(contentLength) + "bytes.")
			}

		} else if inputFilePath != "" {
			if fileInfo, err := os.Stat(inputFilePath); os.IsNotExist(err) {
				inputFilePath = ""
			} else {
				contentLength = int(fileInfo.Size())

				body, err := os.Open(inputFilePath)
				if err != nil {
					return errors.New("Error opening file " + inputFilePath + "\n" + err.Error())
				}
				defer body.Close()

				requestData = make([]byte, contentLength)
				numBytesRead, err := body.Read(requestData)
				if err != nil {
					return errors.New("Error reading input file: " + err.Error())
				}

				if numBytesRead < contentLength {
					return errors.New("Error reading input file: Read " +
						strconv.Itoa(numBytesRead) + " out of " + strconv.Itoa(contentLength) + "bytes.")
				}
			}
		}
	} else if dataOpt != "" {
		return errors.New("Data flag is only valid for POST, PATCH, and PUT requests.")
	}

	requestContentType := ""
	if jsonContentType {
		requestContentType = "application/json"
	} else if contentType != "" {
		requestContentType = contentType
	} else if requestMethod == "POST" || requestMethod == "PATCH" || requestMethod == "PUT" {
		requestContentType = "application/json"
	} else {
		requestContentType = "application/x-www-form-urlencoded"
	}

	accept := "*/*"
	if acceptOpt != "" {
		accept = acceptOpt
	}

	app.InputFilePath = inputFilePath
	app.OutputFilePath = outputFilePath

	app.Request = Request{
		Method:        requestMethod,
		URL:           requestUrl,
		Timeout:       timeout,
		ContentType:   requestContentType,
		Accept:        accept,
		ContentLength: contentLength,
		Body:          requestData,
	}

	return nil
}

// Send HTTP request
func (app *Application) SendRequest() error {
	fmt.Println("Sending request...")

	err := app.loadAndSendHttpRequest()
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

		numBytesWritten, err := file.Write(app.Response.Body)
		if err != nil {
			return errors.New("Error writing json data to file: " + err.Error())
		}

		if numBytesWritten < app.Response.ContentLength {
			return errors.New("Error writing data to output file: Not all data written to file.")
		}

		if err != nil {
			return err
		}
	}

	return nil
}

//
//	Private functions
//

// Create an HTTP request given an app request
func (app *Application) loadAndSendHttpRequest() error {
	requestData := bytes.NewReader(app.Request.Body)
	req, err := http.NewRequest(app.Request.Method, app.Request.URL.String(), requestData)
	if err != nil {
		return errors.New("Error making new request object: " + err.Error())
	}
	if app.Request.ContentType != "" {
		req.Header.Add("Content-Type", app.Request.ContentType)
	}
	if app.Request.Accept != "" {
		req.Header.Add("Accept", app.Request.Accept)
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

	contentType := resp.Header.Get("Content-Type")

	numResponseBytes := len(responseData)
	app.Response = Response{
		ContentType:   contentType,
		ContentLength: numResponseBytes,
		Body:          responseData,
	}
	return nil
}
