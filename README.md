# certmagic-storage-tests

Package tests implements a suite of tests for certmagic.Storage

# Install

    go get github.com/oyato/certmagic-storage-tests

# Usage

_This is package a for testing, so be sure to only import it inside your \_test.go file._

    package storage

    import (
    	tests "github.com/oyato/certmagic-storage-tests"
    	"testing"
    )

    func TestStorage(t *testing.T) {
    	// set up your storage
    	storage := NewInstanceOfYourStorage()
    	// then run the tests on it
    	tests.NewTestSuite(storage).Run(t)
    }

# Note

At this time, it's an exported version of tests used for https://github.com/oyato/certmagic-storage-badger and might be incomplete or unsuitable for testing other storage implementations.
