package respJSON

import (
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
)

type JSONResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func WriteJSONResponse(ctx *fasthttp.RequestCtx, statusCode int, message string, data interface{}) {
	resp := JSONResponse{
		Status:  statusCode,
		Message: message,
		Data:    data,
	}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(`{"status":500,"message":"Internal Server Error"}`))
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(statusCode)
	ctx.SetBody(jsonResp)
}

func WriteJSONError(ctx *fasthttp.RequestCtx, statusCode int, err error, message string) {
	resp := JSONResponse{
		Status:  statusCode,
		Message: fmt.Sprintf("%s: %v", message, err),
	}
	jsonResp, jsonErr := json.Marshal(resp)
	if jsonErr != nil {
		log.Printf("Failed to marshal error JSON: %v", jsonErr)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(`{"status":500,"message":"Internal Server Error"}`))
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(statusCode)
	ctx.SetBody(jsonResp)
}
