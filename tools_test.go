package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

var uploadTest = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:          "allowed no rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    false,
		errorExpected: false,
	},
	{
		name:          "allowed rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    true,
		errorExpected: false,
	},
	{
		name:          "not allowed",
		allowedTypes:  []string{"image/jpeg"},
		renameFile:    false,
		errorExpected: true,
	},
}

func TestTools_RandomString(t *testing.T) {
	var tools Tools
	s := tools.RandomString(10)
	if len(s) != 10 {
		t.Errorf("wrong len, expected: %d, gotted: %d\n", 10, len(s))
	}
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTest {
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

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
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but not received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

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
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools

	err := testTools.CreateDirIfNotExist("./testdata/my-dir")
	if err != nil {
		t.Error(err)
	}

	err = testTools.CreateDirIfNotExist("./testdata/my-dir")
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/my-dir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "now is-+ the{} time+ 321", expected: "now-is-the-time-321", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools

	for _, e := range slugTests {
		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: got error but not expected, %s\n", e.name, err.Error())
		}

		if !e.errorExpected && e.expected != slug {
			t.Errorf("%s: expected %s, gotted: %s\n", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools
	testTools.DownloadStaticFile(rr, req, "./testdata", "img.png", "panorama.png")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "534283" {
		t.Errorf("wrong content length, gotted %s\n", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"panorama.png\"" {
		t.Errorf("wrong content disposition, gotted %s\n", res.Header["Content-Disposition"][0])
	}

	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "good json", json: `{"foo":"bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo":1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json", json: `{"foo":"bar"}{"foo1":"bar1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error json", json: `{"foo":"bar`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field", json: `{"fooo":"bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown field json", json: `{"fooo":"bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{foo:"bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "file too large", json: `{"foo":"bar"}`, errorExpected: true, maxSize: 5, allowUnknown: false},
	{name: "not json", json: `Hello World`, errorExpected: true, maxSize: 1024, allowUnknown: false},
}

func TestTools_JSONRead(t *testing.T) {
	var testTools Tools

	for _, e := range jsonTests {
		testTools.MaxJSONSize = e.maxSize
		testTools.AllowUnknownFields = e.allowUnknown

		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		rr := httptest.NewRecorder()

		err = testTools.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected but not received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected but one received: %s", e.name, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}
	headers := make(http.Header)
	headers.Add("FOO", "BAR")
	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}
	var payload JSONResponse
	dec := json.NewDecoder(rr.Body)
	err = dec.Decode(&payload)
	if err != nil {
		t.Error("error when decoding JSON", err)
	}
	if !payload.Error {
		t.Error("error set to false and it should be true")
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code, expected 503, gotted %d", rr.Code)
	}
}
