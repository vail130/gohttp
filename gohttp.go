/*
	Usage:
		> gohttp 
			[get | head | post | put | patch | delete]
			http://google.com
			[-i | --input /path/to/input/file.json]
			[-o | --output /path/to/output/file.json]
			[-t | --timeout 60]
			[-c | --content-type application/json]

	- Request methods defaults to GET
	- Determines Accept and Content-Type headers from input/output file extensions (txt, html, json, xml)
	- Content type flag overrides extension
	- If no output file specified, saves contents to ~/.gohttp/tmp, and prints only if content is under 100kb

*/
package main

import (
	"os"
	"fmt"
	 "net/url"
)

type Request struct {
	Method string
	Url string
	InputFilePath string
	OutputFilePath string
	Timeout uint32
	ContentType string
}

type Application struct {
	Name string
	Version string
	Args []string
	Request Request
}

func (app *Application) SetupAppDirs() {
	home := os.Getenv("HOME")
	err1 := os.MkdirAll(home + "/.gohttp/requests", 0777)
	if err1 != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory %s\n%s", home + "/.gohttp/requests", err1.Error())
		os.Exit(1)
	}
	err2 := os.MkdirAll(home + "/.gohttp/responses", 0777)
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory %s\n%s", home + "/.gohttp/requests", err2.Error())
		os.Exit(1)
	}
}

func (app *Application) FlagIsActive(flagMap map[string]bool) bool {
	flagIsActive := false
	for i, j := 0, len(app.Args); i < j; i++ {
		if _, present := flagMap[app.Args[i]]; present {
			flagIsActive = true
		}
	}
	return flagIsActive
}

func (app *Application) GetOption(optMap map[string]bool, defaultValue string) string {
	optValue := defaultValue
	for i, j := 0, len(app.Args); i < j; i++ {
		if _, present := optMap[app.Args[i]]; present && len(app.Args) > i {
			defaultValue = app.Args[i + 1]
		}
	}
	return optValue
}

func (app *Application) ParseArgs() {
	fmt.Println("Parsing arguments...")

	if len(app.Args) < 1 {
		fmt.Fprintf(os.Stderr, "No arguments. Try 'gohttp --help' for usage details.")
		os.Exit(1)
	}

	helpFlagMap := map[string]bool {
		"-h": true,
		"--help": true,
	}
	if app.FlagIsActive(helpFlagMap) {
		fmt.Println("> gohttp (get | head | post | put | patch | delete) URL [")
		fmt.Println("	[(-i | --input) /path/to/input/file.json]")
		fmt.Println("	[(-o | --output) /path/to/output/file.json]")
		fmt.Println("	[(-t | --timeout) 0 - 4294967295]")
		fmt.Println("	[(-c | --content-type) application/json]")
		fmt.Println("]")
		os.Exit(0)
	}

	versionFlagMap := map[string]bool {
		"-v": true,
		"--version": true,
	}
	if app.FlagIsActive(versionFlagMap) {
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
	
	if len(app.Args) < urlIndex + 1 {
		fmt.Fprintf(os.Stderr, "Invalid arguments. Try 'gohttp --help' for usage details.")
		os.Exit(1)
	}

	_, err := url.Parse(app.Args[urlIndex])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing URL: %s", err.Error())
	}

	requestUrl := app.Args[urlIndex]

	inputFlagMap := map[string]bool {
		"-i": true,
		"--input": true,
	}
	inputFilePath := app.GetOption(inputFlagMap, "")

	outputFlagMap := map[string]bool {
		"-o": true,
		"--output": true,
	}
	outputFilePath := app.GetOption(outputFlagMap, os.Getenv("HOME") + "/.gohttp/responses")

	contentTypeFlagMap := map[string]bool {
		"-c": true,
		"--content-type": true,
	}
	contentType := app.GetOption(contentTypeFlagMap, "application/json")

	app.Request = Request {
		Method: requestMethod,
		Url: requestUrl,
		InputFilePath: inputFilePath,
		OutputFilePath: outputFilePath,
		Timeout: 0,
		ContentType: contentType,
	}
}

func (app *Application) SendRequest() {
	fmt.Println("Sending request...")

	
}

func main() {
	app := &Application {
		Name: "gohttp",
		Version: "0.1.0",
		Args: os.Args[1:],
	}

	app.SetupAppDirs()
	app.ParseArgs()
	app.SendRequest()
}

