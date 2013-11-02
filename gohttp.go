/*
A command line HTTP request/response management tool in Go.

	Features:
		- Make GET, HEAD, PUT, POST, PATCH, DELETE requests easily
		- Use files as request body
		- Save response body to file
		- Automatic history saving
		- Filter and page history
		- See details and replay requests from history

	Commands:
		[help]
		version
		history
		[REQUESTMETHOD] URL

	History commands:
		history [list] FLAGS
		history detail 1
		history replay 1
		history save 1 /path/to/output/file.json

	HTTP Commands:
		[get] URL FLAGS
		head URL FLAGS
		post URL FLAGS
		put URL FLAGS
		patch URL FLAGS
		delete URL FLAGS

	History Flags:
		(-f | --find) GET
		(-i | --insensitive)
		(-l | --limit) 10
		(-s | --skip) 10

	HTTP Flags:
		(-j | --json)
		(-c | --content-type) application/json
		(-a | --accept) application/json
		(-t | --timeout) 0 - 4294967295
		(-i | --input) /path/to/input/file.json
		(-o | --output) /path/to/output/file.json
		(-d | --data) '{"key": "value"}'
		(-p | --print)
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
