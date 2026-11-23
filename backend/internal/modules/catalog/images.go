package catalog

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mauzo/internal/core/db"
	"mauzo/internal/core/security"
)

// maxImageBytes caps an upload at 5 MB.
const maxImageBytes = 5 << 20

// ImageStore owns image bytes. The catalog records a `path` (store key); the
// implementation decides where bytes live (local disk now, S3 later).
type ImageStore interface {
	// Put writes r under key.
	Put(key string, r io.Reader) error
	// Open returns the bytes stored under key.
	Open(key string) (io.ReadCloser, error)
	// Delete removes key; missing keys are not an error.
	Delete(key string) error
}

// DiskImageStore stores images on the local filesystem under root.
type DiskImageStore struct {
	root string
}

// NewDiskImageStore creates the root dir and returns a DiskImageStore.
func NewDiskImageStore(root string) (*DiskImageStore, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("catalog: image root: %w", err)
	}
	return &DiskImageStore{root: root}, nil
}

// Put writes r to root/key, rejecting paths that escape root.
func (s *DiskImageStore) Put(key string, r io.Reader) error {
	dst, err := s.resolve(key)
	if err != nil {
		return err
	}
	//nolint:gosec // dst is cleaned and confined to root by resolve()
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	//nolint:gosec // dst is cleaned and confined to root by resolve()
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, r)
	return err
}

// Open returns the bytes stored under key.
func (s *DiskImageStore) Open(key string) (io.ReadCloser, error) {
	dst, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	//nolint:gosec // dst is cleaned and confined to root by resolve()
	return os.Open(dst)
}

// Delete removes key; missing keys are not an error.
func (s *DiskImageStore) Delete(key string) error {
	dst, err := s.resolve(key)
	if err != nil {
		return err
	}
	//nolint:gosec // dst is cleaned and confined to root by resolve()
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *DiskImageStore) resolve(key string) (string, error) {
	clean := filepath.Clean(key)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("catalog: bad image key %q", key)
	}
	return filepath.Join(s.root, clean), nil
}

var imageMimes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// AddImage stores an uploaded image for a product. r must be a seekable
// reader limited to maxImageBytes by the caller.
func (s *Service) AddImage(ctx context.Context, tenantID uint64, productPub string, r io.Reader, sniff []byte) (Image, error) {
	p, err := s.repo.ByPublicID(ctx, tenantID, productPub)
	if db.IsNotFound(err) {
		return Image{}, ErrNotFound
	}
	if err != nil {
		return Image{}, err
	}
	ext, ok := imageMimes[http.DetectContentType(sniff)]
	if !ok {
		return Image{}, ErrBadImage
	}
	key := fmt.Sprintf("%d/%d/%s%s", tenantID, p.ID, security.NewPublicID(), ext)
	if err := s.images.Put(key, r); err != nil {
		return Image{}, err
	}
	img := Image{ProductID: p.ID, Path: key}
	existing, err := s.repo.ImagesOf(ctx, p.ID)
	if err != nil {
		return Image{}, err
	}
	img.Sort = len(existing)
	if err := s.repo.AddImage(ctx, &img); err != nil {
		_ = s.images.Delete(key)
		return Image{}, err
	}
	return img, nil
}

// OpenImage returns an image's bytes and its content type.
func (s *Service) OpenImage(ctx context.Context, tenantID, imageID uint64) (io.ReadCloser, string, error) {
	img, err := s.repo.ImageByID(ctx, imageID)
	if db.IsNotFound(err) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if _, err := s.repo.ByID(ctx, tenantID, img.ProductID); err != nil {
		return nil, "", ErrNotFound
	}
	rc, err := s.images.Open(img.Path)
	if err != nil {
		return nil, "", ErrNotFound
	}
	ct := "image/" + strings.TrimPrefix(filepath.Ext(img.Path), ".")
	if strings.HasSuffix(img.Path, ".jpg") {
		ct = "image/jpeg"
	}
	return rc, ct, nil
}

// DeleteImage removes a product's image (row + bytes).
func (s *Service) DeleteImage(ctx context.Context, tenantID, imageID uint64) error {
	img, err := s.repo.ImageByID(ctx, imageID)
	if db.IsNotFound(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := s.repo.ByID(ctx, tenantID, img.ProductID); err != nil {
		return ErrNotFound
	}
	if err := s.repo.DeleteImage(ctx, imageID); err != nil {
		return err
	}
	return s.images.Delete(img.Path)
}
