package plain

import (
	"fmt"
	"io"
	"os"

	"encoding/csv"
)

type options struct {
	credentials map[string]string
}

type Option interface {
	apply(*options) error
}

type optionFunc func(*options) error

func (f optionFunc) apply(o *options) error {
	return f(o)
}

func WithCredentials(credentials map[string]string) Option {
	return optionFunc(func(o *options) error {
		o.credentials = appendCredentials(o.credentials, credentials)
		return nil
	})
}

func WithCredentialsFile(filename string) Option {
	return optionFunc(func(o *options) error {
		if filename == "" {
			return nil
		}
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		credentials, err := credentialsFromCSV(f)
		if err != nil {
			return err
		}
		o.credentials = appendCredentials(o.credentials, credentials)
		return nil
	})
}

func appendCredentials(result map[string]string, credentials map[string]string) map[string]string {
	if result == nil {
		result = make(map[string]string)
	}
	for username, password := range credentials {
		result[username] = password
	}
	return result
}

func credentialsFromCSV(reader io.Reader) (map[string]string, error) {
	credentials := make(map[string]string)

	r := csv.NewReader(reader)
	r.Comment = '#'
	r.FieldsPerRecord = 2
	r.TrimLeadingSpace = true

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) != 2 {
			return nil, fmt.Errorf("csv record username,password expected but got %v", record)
		}
		credentials[record[0]] = record[1]
	}
	return credentials, nil
}
