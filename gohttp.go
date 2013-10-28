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
)

func main() {
	err := application.Start()
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
}
