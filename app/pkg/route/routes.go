package route

import (
	"github.com/Dimoonevs/video-service/app/internal/repo/mysql"
	"github.com/Dimoonevs/video-service/app/internal/service"
	"github.com/Dimoonevs/video-service/app/pkg/respJSON"
	"github.com/valyala/fasthttp"
	"log"
	"strconv"
	"strings"
)

func RequestHandler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.URI().Path())

	if !strings.HasPrefix(path, "/video-service") {
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
		return
	}

	remainingPath := path[len("/video-service"):]

	if strings.HasPrefix(remainingPath, "/check") {
		handleCheck(ctx)
		return
	}
	if strings.HasPrefix(remainingPath, "/upload") {
		if string(ctx.Method()) == "POST" {
			handleUpload(ctx)
		} else {
			respJSON.WriteJSONError(ctx, fasthttp.StatusMethodNotAllowed, nil, "Method not allowed")
		}
		return
	}

	if strings.HasPrefix(remainingPath, "/video") {
		remainingPath := remainingPath[len("/video"):]

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
			respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
		}
		return
	}

	respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
}

func handleUpload(ctx *fasthttp.RequestCtx) {
	isStreamParam := string(ctx.FormValue("is_stream"))
	var isStream bool
	if isStreamParam == "1" || isStreamParam == "true" {
		isStream = true
	} else if isStreamParam == "0" || isStreamParam == "false" {
		isStream = false
	} else {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "Invalid value for is_stream. Expecting true/false or 1/0")
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, err, "Invalid multipart form")
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "No file uploaded")
		return
	}

	if err = service.SaveFile(files, isStream); err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Error saving file")
		return
	}

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "File uploaded successfully", nil)
}

func handleStatusError(ctx *fasthttp.RequestCtx) {
	resp, err := service.GetStatusError()
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Failed to get status error")
		return
	}

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Status error retrieved successfully", resp)
}

func handleVideoErrorsUpdate(ctx *fasthttp.RequestCtx) {
	if err := service.ChangeStatus(); err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Failed to update status error")
		return
	}

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Video errors updated successfully", nil)
}

func handleVideoGetInfo(ctx *fasthttp.RequestCtx) {
	videoStatus := string(ctx.FormValue("status"))
	resp, err := mysql.GetConnection().GetInfoVideos(videoStatus)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Failed to get video info")
		return
	}
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Video info retrieved successfully", resp)
}

func handleDeleteVideoById(ctx *fasthttp.RequestCtx) {
	idVideoStr := string(ctx.FormValue("id"))
	idVideo, err := strconv.Atoi(idVideoStr)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, err, "Invalid video ID")
		log.Printf("Error converting id to int: %v\n", err)
		return
	}
	if err = service.DeleteVideo(idVideo); err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Error deleting video")
		return
	}
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Video deleted successfully", nil)
}

func handlerVegeoGetLinks(ctx *fasthttp.RequestCtx) {
	videoFormatLinksResp, err := mysql.GetConnection().GetVideoLinks()
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Failed to get video links")
		return
	}

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Video links retrieved successfully", videoFormatLinksResp)
}

func handleCheck(ctx *fasthttp.RequestCtx) {
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Service is running", nil)
}
