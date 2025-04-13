package mysql

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Dimoonevs/video-service/app/internal/models"
	"github.com/Dimoonevs/video-service/app/pkg/lib"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"log"
	"mime/multipart"
	"sync"
)

type Storage struct {
	db *sql.DB
}

var (
	mysqlConnectionString = flag.String("SQLConnPassword", "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4,utf8", "DB connection")
	storage               *Storage
	once                  sync.Once
)

func initMySQLConnection() {
	dbConn, err := sql.Open("mysql", *mysqlConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	dbConn.SetMaxIdleConns(0)

	storage = &Storage{
		db: dbConn,
	}
}

func GetConnection() *Storage {
	once.Do(func() {
		initMySQLConnection()
	})

	return storage
}

func (s *Storage) SetFilesData(file *multipart.FileHeader, path string, isStream bool, id int) error {
	query := `
		INSERT INTO files (filename, filepath, is_stream, status, user_id)
		VALUES (?, ?, ?, IF(? = 1, 'conv', 'no_conv'), ?)
	`
	_, err := s.db.Exec(query, file.Filename, path, isStream, isStream, id)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			logrus.Errorf("duplicate entry error: %v", err)
			return err
		} else {
			logrus.Errorf("failed to insert file data: %v", err)
			return err
		}
	}
	return nil
}

func (s *Storage) SetStatusIntoConv(id int) error {
	query := `
		UPDATE files 
		SET status = 'conv' 
		WHERE status = 'error' AND is_stream = 1 and user_id = ?;
	`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (s *Storage) GetInfoVideos(status string, userID, videoID int) ([]*models.InfoVideosResp, error) {
	query := fmt.Sprintf(`
	SELECT id, filename, status, is_stream, filepath, status_ai
	FROM files
	WHERE user_id = %d
`, userID)
	var args []interface{}
	if status != "" {
		query += "AND status = ?"
		args = append(args, status)
	}
	if videoID != 0 {
		query += "AND id = ?"
		args = append(args, videoID)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*models.InfoVideosResp
	for rows.Next() {
		var resp models.InfoVideosResp
		var filepathLocal string
		if err = rows.Scan(&resp.Id, &resp.FileName, &resp.Status, &resp.IsStream, &filepathLocal, &resp.StatusAI); err != nil {
			return nil, err
		}
		if resp.Status != "deleted" {
			resp.FilePath = lib.GetVideoPublicLink(filepathLocal)
		}
		results = append(results, &resp)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Storage) GetInfoVideoById(id int, userID int) (*models.InfoVideosResp, error) {
	query := `
	SELECT id, filename, status, is_stream, filepath, status_ai
	FROM files
	WHERE id = ?
	AND user_id = ?
`
	row := s.db.QueryRow(query, id, userID)

	var videoInfo models.InfoVideosResp
	if err := row.Scan(&videoInfo.Id, &videoInfo.FileName, &videoInfo.Status, &videoInfo.IsStream, &videoInfo.FilePath, &videoInfo.StatusAI); err != nil {
		return nil, err
	}
	return &videoInfo, nil
}

func (s *Storage) DeleteVideo(newFilename string, id, userID int) error {
	query := `
	UPDATE files
	SET status = 'deleted', filename = CONCAT(filename, ?)
	WHERE id = ?
	AND user_id = ?
`
	_, err := s.db.Exec(query, newFilename, id, userID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetVideoLinks(id int) ([]*models.VideoFormatLinksResp, error) {
	query := `
	SELECT 
		f.id AS file_id, 
		f.filename, 
		vf.id AS video_format_id, 
		vf.formats
	FROM 
		files f
	INNER JOIN 
		files_j_video_formats fjvf ON f.id = fjvf.file_id
	INNER JOIN 
		video_formats vf ON fjvf.video_format_id = vf.id
	WHERE 
		f.status = 'done'
	AND f.user_id = ?
`

	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.VideoFormatLinksResp
	var formatsJSON string
	for rows.Next() {
		var resp models.VideoFormatLinksResp
		if err := rows.Scan(&resp.FileId, &resp.Filename, &resp.VideoFormatId, &formatsJSON); err != nil {
			return nil, err
		}
		var videoFormats []models.VideoFormat
		if err := json.Unmarshal([]byte(formatsJSON), &videoFormats); err != nil {
			return nil, fmt.Errorf("failed to unmarshal formats JSON: %w", err)
		}

		resp.Formats = videoFormats

		results = append(results, &resp)

	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
