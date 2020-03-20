package tests

import (
	"github.com/caddyserver/certmagic"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStorage(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "certmagic-storage-tests-")
	if err != nil {
		t.Fatalf("Cannot create temp directory: %s", err)
	}
	defer os.RemoveAll(tempDir)
	fs := &certmagic.FileStorage{
		Path: filepath.Join(tempDir, "filestorage"),
	}
	NewTestSuite(fs).Run(t)
}
