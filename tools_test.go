package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
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

func TestTools_CreateDirIfNotExists(t *testing.T) {
	var tt Tools

	tempDir, err := os.MkdirTemp(".", "testdata-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(tempDir)
	}()

	err = tt.CreateDirIfNotExist(path.Join(tempDir, "/myDir", "mySubdir"))
	if err != nil {
		t.Error(err)
	}
}

func TestTools_Slugify(t *testing.T) {
	tests := []struct {
		name          string
		s             string
		expected      string
		errorExpected bool
	}{
		{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
		{name: "empty string", s: "", expected: "", errorExpected: true},
		{name: "complex string", s: "Now is the time for all GOOD men! + fish & such &^123", expected: "now-is-the-time-for-all-good-men-fish-such-123", errorExpected: false},
		{name: "japanese string", s: "こんにちは世界", expected: "", errorExpected: true},
		{name: "japanese string and roman characters", s: "hello world こんにちは世界", expected: "hello-world", errorExpected: false},
	}

	var tt Tools
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			slug, err := tt.Slugify(tc.s)
			if err != nil && !tc.errorExpected {
				t.Errorf("%s: error received when none expected: %v", tc.name, err)
			}
			if !tc.errorExpected && slug != tc.expected {
				t.Errorf("%s: wrong slung returned. expected %s, but got %s", tc.name, tc.expected, slug)
			}
		})
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/download", nil)

	var tt Tools
	tt.DownloadStaticFile(rr, req, "./testdata", "pic.jpg", "puppy.jpg")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Error("wrong content-length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.jpg\"" {
		t.Errorf("wrong content-disposition")
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}
