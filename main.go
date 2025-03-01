package main

import (
	"flag"
	"fmt"
	"github.com/Dimoonevs/upload-video/app/pkg/route"
	"github.com/valyala/fasthttp"
	"github.com/vharitonsky/iniflags"
)

var (
	port = flag.String("port", "8080", "Port to listen on")
)

func main() {
	iniflags.Parse()

	server := &fasthttp.Server{
		Handler:            route.RequestHandler,
		MaxRequestBodySize: 20 * 104 * 1024 * 1024,
	}

	fmt.Printf("Server is running on %s...\n", *port)
	if err := server.ListenAndServe(fmt.Sprintf(":%s", *port)); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
