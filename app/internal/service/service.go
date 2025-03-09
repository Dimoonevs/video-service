package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/Dimoonevs/video-service/app/internal/repo/mysql"
	"github.com/Dimoonevs/video-service/app/pkg/lib"
	"github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	pathToSave = flag.String("pathToSave", "", "path to save the file")
)

func SaveFile(files []*multipart.FileHeader, isStreams bool, id int) error {
	var skippedFiles []string
	if err := saveFileDiskAndDB(files, &skippedFiles, isStreams, id); err != nil {
		return err
	}
	if len(skippedFiles) > 0 {
		return fmt.Errorf("skipped files: %v", skippedFiles)
	}

	return nil
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

func saveFileImmediately(fileHeader *multipart.FileHeader, savePath string) error {
	tempPath := savePath + "_temp"

	src, err := fileHeader.Open()
	if err != nil {
		logrus.Errorf("open error: %v", err)
		return err
	}
	defer src.Close()

	dir := filepath.Dir(savePath)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		logrus.Errorf("mkdir error: %v", err)
		return err
	}

	dst, err := os.Create(tempPath)
	if err != nil {
		logrus.Errorf("create temp file error: %v", err)
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		logrus.Errorf("copy error: %v", err)
		return err
	}

	err = trimVideo(tempPath, savePath)
	if err != nil {
		logrus.Errorf("ffmpeg trim error: %v", err)
		return err
	}

	err = os.Remove(tempPath)
	if err != nil {
		logrus.Warnf("error remove trim file: %v", err)
	}

	return nil
}
func trimVideo(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-t", "300", "-c", "copy", outputPath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
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
	errChan := make(chan error, len(files)*2)

	for _, file := range files {
		savePath := *pathToSave + hashFilename(id, file.Filename) + "/" + file.Filename

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
			if err := mysql.GetConnection().SetFilesData(file, path, isStreams, id); err != nil {
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

func hashFilename(userID int, filename string) string {
	hasher := md5.New()
	hasher.Write([]byte(fmt.Sprintf("%d_%s", userID, filename))) // Добавляем userID к имени файла
	return hex.EncodeToString(hasher.Sum(nil))
}
