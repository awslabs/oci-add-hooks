package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLossless(t *testing.T) {
	// create hooks with something extra
	// with prestart field with something extra
	configJSON := `{
		"hooks": {
			"prestart": [{"something":"here"}],
			"extra": "things"
			},
			"unique":"words"
	}`
	c := config{
		Hooks: &hooks{},
	}
	h := c.Hooks
	err := c.UnmarshalJSON([]byte(configJSON))
	if err != nil {
		t.Error(err)
	}
	// verify something is in prestart to ensure we marshalled correctly
	if len(h.Prestart) != 1 {
		t.Errorf("expected 1 prestart item but got %d", len(h.Prestart))
	}
	//marshal and verify we have the extra content
	outBytes, err := c.MarshalJSON()
	if err != nil {
		t.Error(err)
	}
	str := string(outBytes)
	// check that marshalled json contains our extra things
	if !strings.Contains(str, "extra") {
		t.Errorf("expected marshalled json to contain 'extra' but it did not: %s", str)
	}
	if !strings.Contains(str, "unique") {
		t.Errorf("expected marshalled json to contain 'unique' but it did not: %s", str)
	}
}

func TestMergeHook(t *testing.T) {
	// Possible inputs
	// both nil
	// one or the other nil
	// boht non-nil
	single := []json.RawMessage{json.RawMessage{}}
	double := []json.RawMessage{json.RawMessage{}, json.RawMessage{}}
	cases := []struct {
		a        []json.RawMessage
		b        []json.RawMessage
		expected []json.RawMessage
	}{
		{
			a:        nil,
			b:        nil,
			expected: nil,
		},
		{
			a:        nil,
			b:        single,
			expected: single,
		},
		{
			a:        single,
			b:        nil,
			expected: single,
		},
		{
			a:        single,
			b:        single,
			expected: double,
		},
	}
	for _, test := range cases {
		res := mergeHook(test.a, test.b)
		if len(test.a)+len(test.b) != len(res) {
			t.Errorf("Expected {%+v} but got {%+v}", test.expected, res)
		}
	}
}

func TestReadHooks(t *testing.T) {
	// possible inputs
	// non-existant file
	// existing file that fails to marshal
	// existing file that successfully marshals
	// file with hooks data only
	// file with hooks data and additional data
	testBundleNoExists := "./testdata/no-file-here.json"
	testBundleBad := "./testdata/bad-spec.json"
	testBundleGood := "./testdata/good-spec.json"
	testCfgGood := "./testdata/good-cfg.json"
	testCfgBad := "./testdata/bad-cfg.json"
	cases := []struct {
		location                    string
		expectedLen                 int
		expectedCreateRuntimeHook   int
		expectedCreateContainerHook int
		expectedStartContainerHook  int
		expectedErr                 bool
	}{
		{
			location:    testBundleNoExists,
			expectedLen: 0,
			expectedErr: true,
		},
		{
			location:    testBundleBad,
			expectedLen: 0,
			expectedErr: true,
		},
		{
			location:    testBundleGood,
			expectedLen: 1,
			expectedErr: false,
		},
		{
			location:    testCfgBad,
			expectedLen: 0,
			expectedErr: true,
		},
		{
			location:                    testCfgGood,
			expectedLen:                 1,
			expectedCreateRuntimeHook:   1,
			expectedCreateContainerHook: 1,
			expectedStartContainerHook:  1,
			expectedErr:                 false,
		},
	}
	for _, test := range cases {
		spec, err := readHooks(test.location)
		// Expected error but got none
		if test.expectedErr && err == nil {
			t.Errorf("Expected test readHooks(%s) to error but it did not", test.location)
		}
		// Expected no error but did get one
		if !test.expectedErr && err != nil {
			t.Errorf("Did not expect readHooks(%s) to error but it did: %s",
				test.location,
				err.Error())
		}
		// Expected no error but spec still nil
		if spec == nil && !test.expectedErr {
			t.Errorf("Expected parsed hooks to be non-nil but it was")
		}
		// If we don't expect an error check if we parsed the expected
		// prestartPath correctly
		if !test.expectedErr &&
			!(spec != nil &&
				spec.Hooks != nil &&
				spec.Hooks.Prestart != nil &&
				len(spec.Hooks.Prestart) == test.expectedLen &&
				len(spec.Hooks.CreateRuntime) == test.expectedCreateRuntimeHook &&
				len(spec.Hooks.CreateContainer) == test.expectedCreateContainerHook &&
				len(spec.Hooks.StartContainer) == test.expectedStartContainerHook) {
			t.Errorf("Did not parse the expected prestart path length. Got %+v expected %d from file %s",
				spec.Hooks,
				test.expectedLen,
				test.location)
		}
	}
}
