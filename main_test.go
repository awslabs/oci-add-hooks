package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRuncError(t *testing.T) {
	exitCmd := exec.Command("/bin/sh", "-c 'exit 2'")
	exitErr := exitCmd.Run()
	cases := []struct {
		err      error
		expected int
	}{
		{
			err:      nil,
			expected: 0,
		},
		{
			err:      errors.New("simple error"),
			expected: 1,
		},
		{
			err:      exitErr,
			expected: 2,
		},
	}
	for _, c := range cases {
		val := processRuncError(c.err)
		if val != c.expected {
			t.Errorf("Expected %d but got %d", c.expected, val)
		}
	}
}

func TestVerifyRuntimePath(t *testing.T) {
	upDir := filepath.Join(os.TempDir(), "made-up-dir-42")
	err := os.Mkdir(upDir, 0700)
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(upDir)
	upFile := filepath.Join(upDir, "made-up-file-that-exists")
	_, err = os.Create(upFile)
	if err != nil {
		t.Error(err)
	}
	noFile := filepath.Join(upDir, "made-up-file-that-does-not-exist")
	// Possible inputs:
	// existing file -- directory -- should err
	// existing file -- file -- should find file
	// non-existant file -- should err
	cases := []struct {
		input        string
		expectedPath string
		expectedErr  error
	}{
		{
			input:        upDir,
			expectedPath: "",
			expectedErr:  errUnableToFindRuntime,
		},
		{
			input:        upFile,
			expectedPath: upFile,
			expectedErr:  nil,
		},
		{
			input:        noFile,
			expectedPath: "",
			expectedErr:  errUnableToFindRuntime,
		},
	}
	for i := range cases {
		str, err := verifyRuntimePath(cases[i].input)
		if str != cases[i].expectedPath || err != cases[i].expectedErr {
			t.Errorf("Expected {%s, %s} but got {%s, %s}",
				cases[i].expectedPath,
				cases[i].expectedErr,
				str,
				err,
			)
		}
	}

}
