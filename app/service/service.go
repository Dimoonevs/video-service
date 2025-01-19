package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
	"upload-video/app/models"
	"upload-video/app/repo/mysql"
	"upload-video/app/utils"
)

var (
	pathToSave = flag.String("pathToSave", "", "path to save the file")
)

func SaveFile(files []*multipart.FileHeader, isStreams bool) error {
	var skippedFiles []string
	saveFileDiskAndDB(files, &skippedFiles, isStreams)
	if len(skippedFiles) > 0 {
		return fmt.Errorf("skipped files: %v", skippedFiles)
	}

	return nil
}

func GetStatusError() ([]*models.StatusErrorResp, error) {
	statErr, err := mysql.GetConnection().GetStatusError()
	if err != nil {
		return nil, err
	}
	return statErr, nil
}
func ChangeStatus() error {
	return mysql.GetConnection().SetStatusIntoConv()
}

func DeleteVideo(id int) error {
	videoInfo, err := mysql.GetConnection().GetInfoVideoById(id)
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
	err = mysql.GetConnection().DeleteVideo(videoInfo.FileName, id)
	if err != nil {
		return err
	}
	return nil
}

func saveFileImmediately(fileHeader *multipart.FileHeader, savePath string) {
	src, err := fileHeader.Open()
	if err != nil {
		log.Fatalf("open error: %d", err)
		return
	}
	defer src.Close()

	dir := filepath.Dir(savePath)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatalf("mkdir error: %d", err)
		return
	}

	dst, err := os.Create(savePath)
	if err != nil {
		log.Fatalf("create error: %d", err)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		log.Fatalf("copy error: %d", err)
		return
	}
}
func deleteParentDir(filePath string) error {
	parentDir := filepath.Dir(filePath)

	err := os.RemoveAll(parentDir)
	if err != nil {
		return err
	}
	return nil
}

func saveFileDiskAndDB(files []*multipart.FileHeader, skippedFiles *[]string, isStreams bool) {
	wg := new(sync.WaitGroup)
	db := mysql.GetConnection()
	for _, file := range files {
		savePath := *pathToSave + hashFilename(file.Filename) + "/" + file.Filename
		if !utils.IsMP4(file.Filename) {
			*skippedFiles = append(*skippedFiles, file.Filename)
			continue
		}
		wg.Add(1)
		go func(file *multipart.FileHeader, path string) {
			defer wg.Done()
			saveFileImmediately(file, path)
		}(file, savePath)

		wg.Add(1)
		go func(file *multipart.FileHeader, path string) {
			defer wg.Done()
			db.SetFilesData(file, path, isStreams)
		}(file, savePath)
	}

	wg.Wait()
}

func hashFilename(filename string) string {
	hasher := md5.New()
	hasher.Write([]byte(filename))
	return hex.EncodeToString(hasher.Sum(nil))
}
