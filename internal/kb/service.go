package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/store"

	"github.com/google/uuid"
)

// MaxFileSize bounds uploaded knowledge files.
const MaxFileSize = 25 << 20

// Service stores, indexes and searches knowledge sources.
type Service struct {
	Store *store.Store
	Box   *secure.Box
	Dir   string // root directory for encrypted blobs
}

func (s *Service) blobPath(tenant int64, key string) string {
	return filepath.Join(s.Dir, "kb", fmt.Sprint(tenant), key+".bin")
}

// Save creates or updates a source. data is the uploaded file (nil keeps
// the existing content); for text sources pass the text as data with an
// empty file name.
func (s *Service) Save(ctx context.Context, src *domain.KnowledgeSource, fileName string, data []byte) error {
	tenant, _ := store.TenantOf(ctx)
	if strings.TrimSpace(src.Title) == "" {
		return errors.New("title is required")
	}
	normalize(src)
	var prev *domain.KnowledgeSource
	if src.ID > 0 {
		p, err := s.Store.KB.Get(ctx, src.ID)
		if err != nil {
			return err
		}
		prev = p
		src.Version, src.StorageKey, src.FileName, src.MimeType = p.Version+1, p.StorageKey, p.FileName, p.MimeType
		src.Size, src.Checksum, src.TextChars, src.Kind = p.Size, p.Checksum, p.TextChars, p.Kind
	}
	var text string
	if data != nil {
		if len(data) > MaxFileSize {
			return fmt.Errorf("file too large (max %d MB)", MaxFileSize>>20)
		}
		var err error
		if text, err = Extract(fileName, data); err != nil {
			return err
		}
		if err := s.writeBlob(tenant, src, fileName, data); err != nil {
			return err
		}
		src.TextChars = len([]rune(text))
	} else if prev == nil {
		return errors.New("content is required")
	}
	src.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	var err error
	if prev == nil {
		src.Version = 1
		err = s.Store.KB.Create(ctx, src)
	} else {
		err = s.Store.KB.Update(ctx, src)
	}
	if err != nil {
		return err
	}
	if text == "" {
		if text, err = s.Text(ctx, src); err != nil {
			return err
		}
	}
	return s.Store.IndexSource(ctx, src.TenantID, src.ID, Chunk(src.Title+"\n"+text, 900))
}

func (s *Service) writeBlob(tenant int64, src *domain.KnowledgeSource, fileName string, data []byte) error {
	sealed, err := s.Box.Seal(tenant, data)
	if err != nil {
		return err
	}
	if src.StorageKey == "" {
		src.StorageKey = uuid.NewString()
	}
	path := s.blobPath(tenant, src.StorageKey)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, sealed, 0o600); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	src.Checksum, src.Size = hex.EncodeToString(sum[:]), int64(len(data))
	if fileName == "" {
		src.Kind, src.FileName, src.MimeType = domain.KBText, "", "text/plain"
	} else {
		src.FileName = filepath.Base(fileName)
		src.MimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
		if src.Kind == "" || src.Kind == domain.KBText {
			src.Kind = domain.KBFile
		}
	}
	return nil
}

// Content decrypts the stored original of a source.
func (s *Service) Content(ctx context.Context, src *domain.KnowledgeSource) ([]byte, error) {
	sealed, err := os.ReadFile(s.blobPath(src.TenantID, src.StorageKey))
	if err != nil {
		return nil, err
	}
	return s.Box.Open(src.TenantID, sealed)
}

// Text returns the extracted text of a source.
func (s *Service) Text(ctx context.Context, src *domain.KnowledgeSource) (string, error) {
	data, err := s.Content(ctx, src)
	if err != nil {
		return "", err
	}
	return Extract(src.FileName, data)
}

// Delete removes a source, its encrypted blob and its index entries.
func (s *Service) Delete(ctx context.Context, id int64) error {
	src, err := s.Store.KB.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.Store.KB.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.Store.DropIndex(ctx, src.TenantID, id)
	if src.StorageKey != "" {
		_ = os.Remove(s.blobPath(src.TenantID, src.StorageKey))
	}
	return nil
}

func normalize(src *domain.KnowledgeSource) {
	if src.Visibility == "" {
		src.Visibility = domain.KBVisibilityAll
	}
	if src.Status == "" {
		src.Status = domain.KBActive
	}
	for _, p := range []*[]string{&src.Tags, &src.Topics, &src.AllowedRoles} {
		if *p == nil {
			*p = []string{}
		}
	}
	for _, p := range []*[]int64{&src.DepartmentIDs, &src.QueueIDs, &src.JobProfileIDs} {
		if *p == nil {
			*p = []int64{}
		}
	}
}
