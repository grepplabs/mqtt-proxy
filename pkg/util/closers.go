package util

import "github.com/hashicorp/go-multierror"

type MultiCloser []func() error

func (c *MultiCloser) Add(closeFunc func() error) {
	if closeFunc == nil {
		return
	}
	*c = append(*c, closeFunc)
}

func (c *MultiCloser) Close() error {
	if len(*c) == 0 {
		return nil
	}
	var result error
	for i := range *c {
		closeFunc := (*c)[len(*c)-1-i]
		if err := closeFunc(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}
