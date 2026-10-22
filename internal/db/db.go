package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"candak/internal/models"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
	mu sync.RWMutex
}

func InitDB(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite works best with 1 open conn for writes

	d := &Database{db: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	return d, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) migrate() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	schema := `
	CREATE TABLE IF NOT EXISTS downloads (
		id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		filename TEXT NOT NULL,
		save_path TEXT NOT NULL,
		total_size INTEGER NOT NULL DEFAULT 0,
		downloaded_size INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'queued',
		resumable INTEGER NOT NULL DEFAULT 0,
		total_chunks INTEGER NOT NULL DEFAULT 1,
		speed INTEGER NOT NULL DEFAULT 0,
		eta INTEGER NOT NULL DEFAULT 0,
		error_message TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);

	CREATE TABLE IF NOT EXISTS chunks (
		id TEXT PRIMARY KEY,
		download_id TEXT NOT NULL REFERENCES downloads(id) ON DELETE CASCADE,
		chunk_index INTEGER NOT NULL,
		start_offset INTEGER NOT NULL,
		end_offset INTEGER NOT NULL,
		current_offset INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending'
	);

	CREATE INDEX IF NOT EXISTS idx_chunks_download_id ON chunks(download_id);

	CREATE TABLE IF NOT EXISTS url_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL UNIQUE,
		filename TEXT NOT NULL DEFAULT '',
		last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		download_count INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_url_history_last_used ON url_history(last_used_at DESC);
	`

	_, err := d.db.Exec(schema)
	return err
}

func (d *Database) InsertDownload(dl *models.Download) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	dl.CreatedAt = now
	dl.UpdatedAt = now

	resumableInt := 0
	if dl.Resumable {
		resumableInt = 1
	}

	query := `
	INSERT INTO downloads (id, url, filename, save_path, total_size, downloaded_size, status, resumable, total_chunks, speed, eta, error_message, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := d.db.Exec(query,
		dl.ID,
		dl.URL,
		dl.Filename,
		dl.SavePath,
		dl.TotalSize,
		dl.DownloadedSize,
		string(dl.Status),
		resumableInt,
		dl.TotalChunks,
		dl.Speed,
		dl.ETA,
		dl.ErrorMessage,
		dl.CreatedAt,
		dl.UpdatedAt,
	)
	return err
}

func (d *Database) UpdateDownloadProgress(id string, downloadedSize int64, speed int64, eta int64, status models.DownloadStatus) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	UPDATE downloads
	SET downloaded_size = ?, speed = ?, eta = ?, status = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := d.db.Exec(query, downloadedSize, speed, eta, string(status), now, id)
	return err
}

func (d *Database) UpdateDownloadStatus(id string, status models.DownloadStatus, errMsg string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	UPDATE downloads
	SET status = ?, error_message = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := d.db.Exec(query, string(status), errMsg, now, id)
	return err
}

func (d *Database) GetDownload(id string) (*models.Download, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, url, filename, save_path, total_size, downloaded_size, status, resumable, total_chunks, speed, eta, error_message, created_at, updated_at
	FROM downloads
	WHERE id = ?
	`
	row := d.db.QueryRow(query, id)

	var dl models.Download
	var resumableInt int
	var statusStr string

	err := row.Scan(
		&dl.ID,
		&dl.URL,
		&dl.Filename,
		&dl.SavePath,
		&dl.TotalSize,
		&dl.DownloadedSize,
		&statusStr,
		&resumableInt,
		&dl.TotalChunks,
		&dl.Speed,
		&dl.ETA,
		&dl.ErrorMessage,
		&dl.CreatedAt,
		&dl.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	dl.Status = models.DownloadStatus(statusStr)
	dl.Resumable = resumableInt == 1

	chunks, err := d.getChunksInternal(id)
	if err == nil {
		dl.Chunks = chunks
	}

	return &dl, nil
}

func (d *Database) GetAllDownloads() ([]*models.Download, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, url, filename, save_path, total_size, downloaded_size, status, resumable, total_chunks, speed, eta, error_message, created_at, updated_at
	FROM downloads
	ORDER BY created_at DESC
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*models.Download
	for rows.Next() {
		var dl models.Download
		var resumableInt int
		var statusStr string

		if err := rows.Scan(
			&dl.ID,
			&dl.URL,
			&dl.Filename,
			&dl.SavePath,
			&dl.TotalSize,
			&dl.DownloadedSize,
			&statusStr,
			&resumableInt,
			&dl.TotalChunks,
			&dl.Speed,
			&dl.ETA,
			&dl.ErrorMessage,
			&dl.CreatedAt,
			&dl.UpdatedAt,
		); err != nil {
			return nil, err
		}

		dl.Status = models.DownloadStatus(statusStr)
		dl.Resumable = resumableInt == 1

		chunks, _ := d.getChunksInternal(dl.ID)
		dl.Chunks = chunks

		downloads = append(downloads, &dl)
	}

	return downloads, nil
}

func (d *Database) DeleteDownload(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM chunks WHERE download_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM downloads WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) SaveChunks(chunks []*models.Chunk) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO chunks (id, download_id, chunk_index, start_offset, end_offset, current_offset, status)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		current_offset = excluded.current_offset,
		status = excluded.status
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range chunks {
		if _, err := stmt.Exec(c.ID, c.DownloadID, c.ChunkIndex, c.StartOffset, c.EndOffset, c.CurrentOffset, string(c.Status)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *Database) UpdateChunkOffset(chunkID string, currentOffset int64, status models.ChunkStatus) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE chunks SET current_offset = ?, status = ? WHERE id = ?`
	_, err := d.db.Exec(query, currentOffset, string(status), chunkID)
	return err
}

func (d *Database) GetChunks(downloadID string) ([]*models.Chunk, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getChunksInternal(downloadID)
}

func (d *Database) getChunksInternal(downloadID string) ([]*models.Chunk, error) {
	query := `
	SELECT id, download_id, chunk_index, start_offset, end_offset, current_offset, status
	FROM chunks
	WHERE download_id = ?
	ORDER BY chunk_index ASC
	`
	rows, err := d.db.Query(query, downloadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*models.Chunk
	for rows.Next() {
		var c models.Chunk
		var statusStr string
		if err := rows.Scan(&c.ID, &c.DownloadID, &c.ChunkIndex, &c.StartOffset, &c.EndOffset, &c.CurrentOffset, &statusStr); err != nil {
			return nil, err
		}
		c.Status = models.ChunkStatus(statusStr)
		chunks = append(chunks, &c)
	}

	return chunks, nil
}

func (d *Database) RecordURLHistory(urlStr, filename string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	INSERT INTO url_history (url, filename, last_used_at, download_count)
	VALUES (?, ?, ?, 1)
	ON CONFLICT(url) DO UPDATE SET
		filename = CASE WHEN excluded.filename != '' THEN excluded.filename ELSE url_history.filename END,
		last_used_at = excluded.last_used_at,
		download_count = url_history.download_count + 1
	`
	_, err := d.db.Exec(query, urlStr, filename, now)
	return err
}

func (d *Database) GetURLHistory(limit int) ([]*models.URLHistoryItem, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT id, url, filename, last_used_at, download_count
	FROM url_history
	ORDER BY last_used_at DESC
	LIMIT ?
	`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.URLHistoryItem
	for rows.Next() {
		var item models.URLHistoryItem
		if err := rows.Scan(&item.ID, &item.URL, &item.Filename, &item.LastUsedAt, &item.DownloadCount); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

func (d *Database) ClearURLHistory() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("DELETE FROM url_history")
	return err
}
