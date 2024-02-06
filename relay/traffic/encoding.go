package traffic

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func GetContentEncoding(request *http.Request) (string, error) {
	// NOTE: This is a workaround for a bug in post-Go 1.17. See golang.org/issue/25192.
	// Our algorithm differs from the logic of AllowQuerySemicolons by replacing semicolons with encoded semicolons instead
	// of with ampersands. This is because we want to preserve the original query string as much as possible.
	if strings.Contains(request.URL.RawQuery, ";") {
		request.URL.RawQuery = strings.ReplaceAll(request.URL.RawQuery, ";", "%3B") // Replace semicolons with encoded semicolons.
	}

	queryParams, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return "", err
	}

	// request query parameter takes precedence over request header
	encoding := queryParams.Get("ContentEncoding")
	if encoding == "" {
		encoding = request.Header.Get("Content-Encoding")
	}
	return encoding, nil
}

// WrapReader checks if the request Content-Encoding or request query parameter indicates gzip compression.
// If so, it returns a gzip.Reader that decompresses the content.
func WrapReader(request *http.Request, encoding string) (io.ReadCloser, error) {
	if request.Body == nil {
		return nil, nil
	}

	switch encoding {
	case "gzip":
		// Create a new gzip.Reader to decompress the request body
		return gzip.NewReader(request.Body)
	default:
		// If the content is not gzip-compressed, return the original request body
		return request.Body, nil
	}
}

func EncodeData(data []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "gzip":
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
	default:
		// identity encoding
		return data, nil
	}
}

func DecodeData(data []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}

		decodedData, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		return decodedData, nil
	default:
		// identity encoding
		return data, nil
	}
}
