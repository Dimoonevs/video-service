package mysql

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"mime/multipart"
	"sync"
	"upload-video/app/models"
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

func (s *Storage) SetFilesData(file *multipart.FileHeader, path string, isStream bool) {
	query := `
		INSERT INTO files (filename, filepath, is_stream, status)
		VALUES (?, ?, ?, IF(? = 1, 'conv', 'no_conv'))
	`
	_, err := s.db.Exec(query, file.Filename, path, isStream, isStream)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			log.Printf("duplicate entry error: %v", err)
		} else {
			log.Fatalf("failed to insert file data: %v", err)
		}
		return
	}
}

func (s *Storage) GetStatusError() ([]*models.StatusErrorResp, error) {
	query := `
		SELECT id, filename 
		FROM files 
		WHERE status = 'error' AND is_stream = 1;
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.StatusErrorResp

	for rows.Next() {
		var resp models.StatusErrorResp
		if err := rows.Scan(&resp.Id, &resp.FileName); err != nil {
			return nil, err
		}
		results = append(results, &resp)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *Storage) SetStatusIntoConv() error {
	query := `
		UPDATE files 
		SET status = 'conv' 
		WHERE status = 'error' AND is_stream = 1;
	`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (s *Storage) GetInfoVideos(status string) ([]*models.InfoVideosResp, error) {
	query := `
	SELECT id, filename, status, is_stream
	FROM files
`
	var args []interface{}
	if status != "" {
		query += "WHERE status = ?"
		args = append(args, status)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*models.InfoVideosResp
	for rows.Next() {
		var resp models.InfoVideosResp
		if err := rows.Scan(&resp.Id, &resp.FileName, &resp.Status, &resp.IsStream); err != nil {
			return nil, err
		}
		results = append(results, &resp)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Storage) GetInfoVideoById(id int) (*models.InfoVideosResp, error) {
	query := `
	SELECT id, filename, status, is_stream, filepath
	FROM files
	WHERE id = ?
`
	row := s.db.QueryRow(query, id)

	var videoInfo models.InfoVideosResp
	if err := row.Scan(&videoInfo.Id, &videoInfo.FileName, &videoInfo.Status, &videoInfo.IsStream, &videoInfo.FilePath); err != nil {
		return nil, err
	}
	return &videoInfo, nil
}

func (s *Storage) DeleteVideo(newFilename string, id int) error {
	query := `
	UPDATE files
	SET status = 'deleted', filename = CONCAT(filename, ?)
	WHERE id = ?
`
	_, err := s.db.Exec(query, newFilename, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetVideoLinks() ([]*models.VideoFormatLinksResp, error) {
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
`

	rows, err := s.db.Query(query)
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
