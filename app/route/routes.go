package routes

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
	"log"
	"strconv"
	"strings"
	"upload-video/app/repo/mysql"
	"upload-video/app/service"
)

func RequestHandler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.URI().Path())

	if strings.HasPrefix(path, "/upload") {
		if string(ctx.Method()) == "POST" {
			handleUpload(ctx)
		} else {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			ctx.SetBody([]byte("Method not allowed"))
		}
		return
	}

	if strings.HasPrefix(path, "/video") {
		remainingPath := path[len("/video"):]

		switch remainingPath {
		case "/delete":
			handleDeleteVideoById(ctx)
		case "/errors/update":
			handleVideoErrorsUpdate(ctx)
		case "/errors":
			handleStatusError(ctx)
		case "/links":
			handlerVegeoGetLinks(ctx)
		case "":
			handleVideoGetInfo(ctx)
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			ctx.SetBody([]byte("Endpoint not found2"))
		}
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetBody([]byte("Endpoint not found1"))
}

func handleUpload(ctx *fasthttp.RequestCtx) {

	isStreamParam := string(ctx.FormValue("is_stream"))
	var isStream bool
	if isStreamParam == "1" || isStreamParam == "true" {
		isStream = true
	} else if isStreamParam == "0" || isStreamParam == "false" {
		isStream = false
	} else {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBody([]byte("Invalid value for is_stream. Expecting true/false or 1/0"))
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBody([]byte("Invalid multipart form"))
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBody([]byte("No file uploaded"))
		return
	}

	if err = service.SaveFile(files, isStream); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("File uploaded successfully"))

}

func handleStatusError(ctx *fasthttp.RequestCtx) {
	resp, err := service.GetStatusError()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte("Failed to get status error"))
		return
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte("Failed to marshal response"))
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(jsonResp)

}

func handleVideoErrorsUpdate(ctx *fasthttp.RequestCtx) {
	if err := service.ChangeStatus(); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte("Failed to update status error"))
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("File updates successfully"))
}

func handleVideoGetInfo(ctx *fasthttp.RequestCtx) {
	videoStatus := string(ctx.FormValue("status"))
	resp, err := mysql.GetConnection().GetInfoVideos(videoStatus)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return
	}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte("Failed to marshal response"))
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(jsonResp)
}

func handleDeleteVideoById(ctx *fasthttp.RequestCtx) {
	idVideoStr := string(ctx.FormValue("id"))
	idVideo, err := strconv.Atoi(idVideoStr)
	if err != nil {
		log.Printf("Error converting id to int: %v\n", err)
		return
	}
	if err = service.DeleteVideo(idVideo); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("Video deleted successfully"))
}
func handlerVegeoGetLinks(ctx *fasthttp.RequestCtx) {
	videoFormatLinksResp, err := mysql.GetConnection().GetVideoLinks()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return
	}
	jsonResp, err := json.Marshal(videoFormatLinksResp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte("Failed to marshal response"))
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(jsonResp)
}
