package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
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
	t.Parallel()
	var tt Tools

	s := tt.RandomString(10)
	if len(s) != 10 {
		t.Errorf("expected len of %d, got %d", 10, len(s))
	}
}

func TestTools_UploadFiles(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TODO: add check for the error message itself
var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly-formatted json", json: `{"foo": }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "1"}{"alpha": "beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "synatax error in json", json: `{"foo": 1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unkown field in json", json: `{"foooo": "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unkown fields in json", json: `{"foooo": "1"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: true},
	{name: "not json", json: `Hello, world!`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	t.Parallel()
	var tt Tools

	for _, tc := range jsonTests {
		// set the max file size
		tt.MaxJSONSize = tc.maxSize

		// allow/disallow unknown fields
		tt.AllowUnknownFields = tc.allowUnknown

		// declare a var to read the decode json into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a request with the body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(tc.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		rr := httptest.NewRecorder()

		err = tt.ReadJSON(rr, req, &decodedJSON)
		if tc.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", tc.name)
		}

		if !tc.errorExpected && err != nil {
			t.Errorf("%s: error not expected but one recieved: %s", tc.name, err.Error())
		}
		req.Body.Close()

	}
}

func TestTools_WriteJSON(t *testing.T) {
	t.Parallel()
	var tt Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := tt.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Fatal("failed to write json", err)
	}

}

func TestTools_ErrorJSON(t *testing.T) {
	t.Parallel()
	var tt Tools
	rr := httptest.NewRecorder()
	err := tt.ErrorJSON(rr, errors.New("some error"), http.StatusInternalServerError)
	if err != nil {
		t.Fatal(err)
	}

	var payload JSONResponse
	err = json.NewDecoder(rr.Body).Decode(&payload)
	if err != nil {
		t.Fatal("error decoding json:", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
