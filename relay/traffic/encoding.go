package traffic

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Encoding int

const (
	Unsupported Encoding = iota
	Identity
	Gzip
)

func GetContentEncoding(request *http.Request) (Encoding, error) {
	// NOTE: This is a workaround for a bug in post-Go 1.17. See golang.org/issue/25192.
	// Our algorithm differs from the logic of AllowQuerySemicolons by replacing semicolons with encoded semicolons instead
	// of with ampersands. This is because we want to preserve the original query string as much as possible.
	if strings.Contains(request.URL.RawQuery, ";") {
		request.URL.RawQuery = strings.ReplaceAll(request.URL.RawQuery, ";", "%3B") // Replace semicolons with encoded semicolons.
	}

	queryParams, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return Unsupported, err
	}

	// request query parameter takes precedence over request header
	encoding := queryParams.Get("ContentEncoding")
	if encoding == "" {
		encoding = request.Header.Get("Content-Encoding")
	}

	switch encoding {
	case "gzip":
		return Gzip, nil
	case "":
		return Identity, nil
	default:
		return Unsupported, fmt.Errorf("unsupported encoding: %v", encoding)
	}
}

// WrapReader returns a wrapped request.Body for the encoding provided.
func WrapReader(request *http.Request, encoding Encoding) (io.ReadCloser, error) {
	if request.Body == nil {
		return nil, nil
	}

	switch encoding {
	case Gzip:
		// Create a new gzip.Reader to decompress the request body
		return gzip.NewReader(request.Body)
	case Identity:
		// If the content is not gzip-compressed, return the original request body
		return request.Body, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %v", encoding)
	}
}

func EncodeData(data []byte, encoding Encoding) ([]byte, error) {
	switch encoding {
	case Gzip:
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)

		_, err := gz.Write(data)
		if err != nil {
			return nil, err
		}

		err = gz.Close()
		if err != nil {
			return nil, err
		}

		compressedData := buf.Bytes()
		return compressedData, nil
	case Identity:
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %v", encoding)
	}
}

func DecodeData(data []byte, encoding Encoding) ([]byte, error) {
	switch encoding {
	case Gzip:
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}

		decodedData, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		return decodedData, nil
	case Identity:
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %v", encoding)
	}
}
