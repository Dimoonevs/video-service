package route

import (
	"fmt"
	"github.com/Dimoonevs/user-service/app/pkg/jwt"
	"github.com/Dimoonevs/video-service/app/internal/repo/mysql"
	"github.com/Dimoonevs/video-service/app/internal/service"
	"github.com/Dimoonevs/video-service/app/pkg/respJSON"
	"github.com/valyala/fasthttp"
	"log"
	"strconv"
	"strings"
)

func RequestHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) == fasthttp.MethodOptions {
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}

	path := string(ctx.URI().Path())

	if !strings.HasPrefix(path, "/video-service") {
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
		return
	}

	jwt.JWTMiddleware(func(ctx *fasthttp.RequestCtx) {
		handleRoutes(ctx, path)
	})(ctx)
}

func handleRoutes(ctx *fasthttp.RequestCtx, path string) {
	remainingPath := path[len("/video-service"):]

	switch {
	case strings.HasPrefix(remainingPath, "/check"):
		handleCheck(ctx)
	case strings.HasPrefix(remainingPath, "/upload"):
		if string(ctx.Method()) == "POST" {
			handleUpload(ctx)
		} else {
			respJSON.WriteJSONError(ctx, fasthttp.StatusMethodNotAllowed, nil, "Method not allowed")
		}
	case strings.HasPrefix(remainingPath, "/video"):
		handleVideoRoutes(ctx, remainingPath[len("/video"):])
	default:
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
	}
}

func handleVideoRoutes(ctx *fasthttp.RequestCtx, remainingPath string) {
	switch remainingPath {
	case "/delete":
		handleDeleteVideoById(ctx)
	case "/errors/update":
		handleVideoErrorsUpdate(ctx)
	case "/links":
		handlerVideoGetLinks(ctx)
	case "":
		handleVideoGetInfo(ctx)
	default:
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
	}
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

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
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

	service.SaveFile(files, isStream, userID)

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "File uploaded in process", nil)
}

func handleVideoErrorsUpdate(ctx *fasthttp.RequestCtx) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
		return
	}
	if err := mysql.GetConnection().SetStatusIntoConv(userID); err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Failed to update status error")
		return
	}

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Video errors updated successfully", nil)
}

func handleVideoGetInfo(ctx *fasthttp.RequestCtx) {
	videoStatus := string(ctx.FormValue("status"))
	videoID := ctx.QueryArgs().GetUintOrZero("id")
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
		return
	}
	resp, err := mysql.GetConnection().GetInfoVideos(videoStatus, userID, videoID)
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
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
		return
	}
	if err = service.DeleteVideo(idVideo, userID); err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Error deleting video")
		return
	}
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Video deleted successfully", nil)
}

func handlerVideoGetLinks(ctx *fasthttp.RequestCtx) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
		return
	}
	videoFormatLinksResp, err := mysql.GetConnection().GetVideoLinks(userID)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Failed to get video links")
		return
	}

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Video links retrieved successfully", videoFormatLinksResp)
}

func handleCheck(ctx *fasthttp.RequestCtx) {
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusOK, "Service is running", nil)
}

func getUserIDFromContext(ctx *fasthttp.RequestCtx) (int, error) {
	userIDValue := ctx.UserValue("userID")
	userIDFloat, ok := userIDValue.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid userID format: %f", userIDFloat)
	}

	return int(userIDFloat), nil
}
