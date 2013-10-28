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
	"./application"
	"log"
	"os"
	"path"
)

func main() {
	home := os.Getenv("HOME")
	requestPath := path.Join(home, ".gohttp/requests")
	responsePath := path.Join(home, ".gohttp/responses")

	app := &application.Application{
		Name:         "gohttp",
		Version:      "0.1.0",
		Args:         os.Args[1:],
		RequestPath:  requestPath,
		ResponsePath: responsePath,
	}

	err := app.SetupAppDirs()
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}

	err = app.ParseArgs()
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}

	err = app.SendRequest()
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}

	err = app.SaveResponse()
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}

}
