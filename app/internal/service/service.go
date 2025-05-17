package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/Dimoonevs/video-service/app/internal/models"
	"github.com/Dimoonevs/video-service/app/internal/repo/mysql"
	"github.com/Dimoonevs/video-service/app/pkg/lib"
	"github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	pathToSave = flag.String("pathToSave", "", "path to save the file")
)

func SaveFile(files []*multipart.FileHeader, isStreams bool, id int) {
	var skippedFiles []string
	if err := saveFileDiskAndDB(files, &skippedFiles, isStreams, id); err != nil {
		logrus.Errorf("unable to save file: %v", err)
	}
	if len(skippedFiles) > 0 {
		logrus.Errorf("skipped files: %v", skippedFiles)
	}
}

func DeleteVideo(id int, userID int) error {
	videoInfo, err := mysql.GetConnection().GetInfoVideoById(id, userID)
	if err != nil {
		return err
	}
	if videoInfo.Status == "deleted" {
		return errors.New("video is delete")
	}
	if err = deleteParentDir(videoInfo.FilePath); err != nil {
		return err
	}
	videoInfo.FileName = fmt.Sprintf("_%s_%d", "deleted", id)
	err = mysql.GetConnection().DeleteVideo(videoInfo.FileName, id, userID)
	if err != nil {
		return err
	}
	return nil
}

func saveBytesToDisk(data []byte, savePath string) error {
	dir := filepath.Dir(savePath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		logrus.Errorf("mkdir error: %v", err)
		return err
	}

	dst, err := os.Create(savePath)
	if err != nil {
		logrus.Errorf("create file error: %v", err)
		return err
	}
	defer dst.Close()

	_, err = dst.Write(data)
	if err != nil {
		logrus.Errorf("write error: %v", err)
		return err
	}

	return nil
}

func deleteParentDir(filePath string) error {
	parentDir := filepath.Dir(filePath)

	err := os.RemoveAll(parentDir)
	if err != nil {
		return err
	}
	return nil
}

func saveFileDiskAndDB(files []*multipart.FileHeader, skippedFiles *[]string, isStreams bool, id int) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		savePath := *pathToSave + hashFilename(id, file.Filename) + "/" + file.Filename

		if !lib.IsMP4(file.Filename) {
			*skippedFiles = append(*skippedFiles, file.Filename)
			continue
		}

		src, err := file.Open()
		if err != nil {
			logrus.Errorf("failed to open file %s: %v", file.Filename, err)
			continue
		}
		defer src.Close()

		fileBytes, err := io.ReadAll(src)
		if err != nil {
			logrus.Errorf("failed to read file %s: %v", file.Filename, err)
			continue
		}

		filesId, err := mysql.GetConnection().SetFilesData(file.Filename, savePath, isStreams, id)
		if err != nil {
			logrus.Errorf("SetFilesData failed for %s: %v", file.Filename, err)
			continue
		}

		wg.Add(1)
		go func(path string, data []byte, filesId int, filename string) {
			defer wg.Done()
			if err := saveBytesToDisk(data, path); err != nil {
				mysql.GetConnection().SetStatusByFilesID(filesId, models.StatusLoadError)
				errChan <- fmt.Errorf("error while saving file %s: %w", filename, err)
				return
			}

			mysql.GetConnection().SetStatusByFilesID(filesId, models.StatusNoConv)
			logrus.Infof("file %s saved successfully", filename)
		}(savePath, fileBytes, filesId, file.Filename)
	}

	wg.Wait()
	close(errChan)

	var errStrings []string
	for err := range errChan {
		errStrings = append(errStrings, err.Error())
	}

	if len(errStrings) > 0 {
		return errors.New(strings.Join(errStrings, "; "))
	}
	return nil
}

func hashFilename(userID int, filename string) string {
	hasher := md5.New()
	hasher.Write([]byte(fmt.Sprintf("%d_%s", userID, filename)))
	return hex.EncodeToString(hasher.Sum(nil))
}
