package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var tt Tools

	s := tt.RandomString(10)
	if len(s) != 10 {
		t.Errorf("expected len of %d, got %d", 10, len(s))
	}
}

func TestTools_UploadFiles(t *testing.T) {
	tests := []struct {
		name          string
		allowedTypes  []string
		renameFile    bool
		errorExpected bool
	}{
		{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
		{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
		{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// set up a pipe to avoid buffering
			pr, pw := io.Pipe()
			writer := multipart.NewWriter(pw)
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				defer writer.Close()

				// create the form data field 'file'
				part, err := writer.CreateFormFile("file", "./testdata/img.png")
				if err != nil {
					t.Error(err)
				}

				f, err := os.Open("./testdata/img.png")
				if err != nil {
					t.Error(err)
				}
				defer f.Close()

				img, _, err := image.Decode(f)
				if err != nil {
					t.Error("error decoding image:", err)
				}

				err = png.Encode(part, img)
				if err != nil {
					t.Error("error decoding image:", err)
				}

			}()

			// read from the pipe which receives data
			req := httptest.NewRequest("POST", "/", pr)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			var tt Tools
			tt.AllowedFileTypes = tc.allowedTypes

			uploadedFiles, err := tt.UploadFiles(req, "./testdata/uploads/", tc.renameFile)
			if err != nil && !tc.errorExpected {
				t.Error(err)
			}

			if !tc.errorExpected {
				if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
					t.Errorf("%s: expected file to exist: %s", tc.name, err)
				}

				// clean up
				_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
			}

			if tc.errorExpected && err == nil {
				t.Errorf("%s: error expected but none received", tc.name)
			}

			wg.Wait()
		})
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	tests := []struct {
		name          string
		allowedTypes  []string
		renameFile    bool
		errorExpected bool
	}{
		{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
		{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
		{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// set up a pipe to avoid buffering
			pr, pw := io.Pipe()
			writer := multipart.NewWriter(pw)

			go func() {

				defer writer.Close()

				// create the form data field 'file'
				part, err := writer.CreateFormFile("file", "./testdata/img.png")
				if err != nil {
					t.Error(err)
				}

				f, err := os.Open("./testdata/img.png")
				if err != nil {
					t.Error(err)
				}
				defer f.Close()

				img, _, err := image.Decode(f)
				if err != nil {
					t.Error("error decoding image:", err)
				}

				err = png.Encode(part, img)
				if err != nil {
					t.Error("error decoding image:", err)
				}

			}()

			// read from the pipe which receives data
			req := httptest.NewRequest("POST", "/", pr)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			var tt Tools

			uploadedFiles, err := tt.UploadOneFile(req, "./testdata/uploads/", true)
			if err != nil {
				t.Error(err)
			}

			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
				t.Errorf("expected file to exist: %s", err)
			}

			// clean up
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))

		})
	}
}
