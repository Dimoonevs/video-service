package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/Dimoonevs/video-service/app/internal/lib"
	"github.com/Dimoonevs/video-service/app/internal/models"
	"github.com/Dimoonevs/video-service/app/internal/repo/mysql"
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

func SaveFile(files []*multipart.FileHeader, isStreams bool) error {
	var skippedFiles []string
	if err := saveFileDiskAndDB(files, &skippedFiles, isStreams); err != nil {
		return err
	}
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

func saveFileImmediately(fileHeader *multipart.FileHeader, savePath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		logrus.Errorf("open error: %d", err)
		return err
	}
	defer src.Close()

	dir := filepath.Dir(savePath)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		logrus.Errorf("mkdir error: %d", err)
		return err
	}

	dst, err := os.Create(savePath)
	if err != nil {
		logrus.Errorf("create error: %v", err)
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		logrus.Errorf("copy error: %v", err)
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

func saveFileDiskAndDB(files []*multipart.FileHeader, skippedFiles *[]string, isStreams bool) error {
	var wg sync.WaitGroup
	db := mysql.GetConnection()
	errChan := make(chan error, len(files)*2)

	for _, file := range files {
		savePath := *pathToSave + hashFilename(file.Filename) + "/" + file.Filename

		if !lib.IsMP4(file.Filename) {
			*skippedFiles = append(*skippedFiles, file.Filename)
			continue
		}

		wg.Add(1)
		go func(file *multipart.FileHeader, path string) {
			defer wg.Done()
			if err := saveFileImmediately(file, path); err != nil {
				errChan <- fmt.Errorf("ошибка при сохранении файла %s: %w", file.Filename, err)
			}
		}(file, savePath)

		wg.Add(1)
		go func(file *multipart.FileHeader, path string) {
			defer wg.Done()
			if err := db.SetFilesData(file, path, isStreams); err != nil {
				errChan <- fmt.Errorf("ошибка при записи в БД файла %s: %w", file.Filename, err)
			}
		}(file, savePath)
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

func hashFilename(filename string) string {
	hasher := md5.New()
	hasher.Write([]byte(filename))
	return hex.EncodeToString(hasher.Sum(nil))
}
