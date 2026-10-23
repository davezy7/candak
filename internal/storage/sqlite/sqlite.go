package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"candak/internal/models"
	"candak/internal/storage"

	_ "modernc.org/sqlite"
)

// Storage provides a thread-safe SQLite implementation of storage.Repository.
type Storage struct {
	db *sql.DB
	mu sync.RWMutex
}

var _ storage.Repository = (*Storage)(nil)

// New creates and initializes a new SQLite repository with schema migrations.
func New(dbPath string) (*Storage, error) {
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite connection: %w", err)
	}

	// SQLite operates most reliably with a single writer connection pool
	conn.SetMaxOpenConns(1)

	s := &Storage{db: conn}
	if err := s.migrate(context.Background()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to run database migration: %w", err)
	}

	return s, nil
}

// Close closes the underlying database pool.
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

func (s *Storage) migrate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// InsertDownload saves a new download record.
func (s *Storage) InsertDownload(ctx context.Context, dl *models.Download) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	dl.CreatedAt = now
	dl.UpdatedAt = now

	resumableInt := 0
	if dl.Resumable {
		resumableInt = 1
	}

	query := `
	INSERT INTO downloads (
		id, url, filename, save_path, total_size, downloaded_size, status,
		resumable, total_chunks, speed, eta, error_message, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
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

// UpdateDownloadProgress updates byte counter, transfer speed, and ETA.
func (s *Storage) UpdateDownloadProgress(ctx context.Context, id string, downloadedSize int64, speed int64, eta int64, status models.DownloadStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	UPDATE downloads
	SET downloaded_size = ?, speed = ?, eta = ?, status = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, downloadedSize, speed, eta, string(status), now, id)
	return err
}

// UpdateDownloadStatus updates the download's operational status and error text.
func (s *Storage) UpdateDownloadStatus(ctx context.Context, id string, status models.DownloadStatus, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	UPDATE downloads
	SET status = ?, error_message = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, string(status), errMsg, now, id)
	return err
}

// GetDownload retrieves a single download by ID, including its associated chunks.
func (s *Storage) GetDownload(ctx context.Context, id string) (*models.Download, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, url, filename, save_path, total_size, downloaded_size, status,
	       resumable, total_chunks, speed, eta, error_message, created_at, updated_at
	FROM downloads
	WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

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
		if err == sql.ErrNoRows {
			return nil, models.ErrDownloadNotFound
		}
		return nil, err
	}

	dl.Status = models.DownloadStatus(statusStr)
	dl.Resumable = resumableInt == 1

	chunks, err := s.getChunksInternal(ctx, id)
	if err == nil {
		dl.Chunks = chunks
	}

	return &dl, nil
}

// GetAllDownloads retrieves all downloads ordered chronologically descending.
func (s *Storage) GetAllDownloads(ctx context.Context) ([]*models.Download, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, url, filename, save_path, total_size, downloaded_size, status,
	       resumable, total_chunks, speed, eta, error_message, created_at, updated_at
	FROM downloads
	ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*models.Download
	dlMap := make(map[string]*models.Download)
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

		downloads = append(downloads, &dl)
		dlMap[dl.ID] = &dl
	}
	_ = rows.Close()

	// Load all chunk records in one query instead of N+1 per download
	if len(downloads) > 0 {
		chunkRows, err := s.db.QueryContext(ctx, `
		SELECT id, download_id, chunk_index, start_offset, end_offset, current_offset, status
		FROM chunks
		ORDER BY download_id, chunk_index ASC
		`)
		if err == nil {
			defer chunkRows.Close()
			for chunkRows.Next() {
				var c models.Chunk
				var statusStr string
				if err := chunkRows.Scan(&c.ID, &c.DownloadID, &c.ChunkIndex, &c.StartOffset, &c.EndOffset, &c.CurrentOffset, &statusStr); err == nil {
					c.Status = models.ChunkStatus(statusStr)
					if targetDL, ok := dlMap[c.DownloadID]; ok {
						targetDL.Chunks = append(targetDL.Chunks, &c)
					}
				}
			}
		}
	}

	return downloads, nil
}

// DeleteDownload removes a download record and cascades chunk deletion.
func (s *Storage) DeleteDownload(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM chunks WHERE download_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM downloads WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

// SaveChunks transactionally inserts or updates a slice of chunk records.
func (s *Storage) SaveChunks(ctx context.Context, chunks []*models.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
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
		if _, err := stmt.ExecContext(ctx, c.ID, c.DownloadID, c.ChunkIndex, c.StartOffset, c.EndOffset, c.CurrentOffset, string(c.Status)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateChunkOffset persists a chunk's progress offset and status.
func (s *Storage) UpdateChunkOffset(ctx context.Context, chunkID string, currentOffset int64, status models.ChunkStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `UPDATE chunks SET current_offset = ?, status = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, currentOffset, string(status), chunkID)
	return err
}

// GetChunks retrieves all chunks assigned to a download ID in ascending index order.
func (s *Storage) GetChunks(ctx context.Context, downloadID string) ([]*models.Chunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getChunksInternal(ctx, downloadID)
}

func (s *Storage) getChunksInternal(ctx context.Context, downloadID string) ([]*models.Chunk, error) {
	query := `
	SELECT id, download_id, chunk_index, start_offset, end_offset, current_offset, status
	FROM chunks
	WHERE download_id = ?
	ORDER BY chunk_index ASC
	`
	rows, err := s.db.QueryContext(ctx, query, downloadID)
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

// RecordURLHistory inserts or increments the usage of a URL.
func (s *Storage) RecordURLHistory(ctx context.Context, urlStr, filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	INSERT INTO url_history (url, filename, last_used_at, download_count)
	VALUES (?, ?, ?, 1)
	ON CONFLICT(url) DO UPDATE SET
		filename = CASE WHEN excluded.filename != '' THEN excluded.filename ELSE url_history.filename END,
		last_used_at = excluded.last_used_at,
		download_count = url_history.download_count + 1
	`
	_, err := s.db.ExecContext(ctx, query, urlStr, filename, now)
	return err
}

// GetURLHistory retrieves the most recently used URLs up to the specified limit.
func (s *Storage) GetURLHistory(ctx context.Context, limit int) ([]*models.URLHistoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT id, url, filename, last_used_at, download_count
	FROM url_history
	ORDER BY last_used_at DESC
	LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
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

// ClearURLHistory purges all recorded URL history records.
func (s *Storage) ClearURLHistory(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM url_history")
	return err
}
