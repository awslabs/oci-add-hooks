package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	lossless "github.com/joeshaw/json-lossless"
)

type config struct {
	lossless.JSON `json:"-"`

	Hooks *hooks `json:"hooks"`
}

func (c *config) UnmarshalJSON(data []byte) error {
	return c.JSON.UnmarshalJSON(c, data)
}

func (c *config) MarshalJSON() ([]byte, error) {
	return c.JSON.MarshalJSON(c)
}

type hooks struct {
	lossless.JSON `json:"-"`

	Prestart        []json.RawMessage `json:"prestart"`
	CreateRuntime   []json.RawMessage `json:"createRuntime"`
	CreateContainer []json.RawMessage `json:"createContainer"`
	StartContainer  []json.RawMessage `json:"startContainer"`
	Poststart       []json.RawMessage `json:"poststart"`
	Poststop        []json.RawMessage `json:"poststop"`
}

func (h *hooks) UnmarshalJSON(data []byte) error {
	return h.JSON.UnmarshalJSON(h, data)
}

func (h *hooks) MarshalJSON() ([]byte, error) {
	return h.JSON.MarshalJSON(h)
}

func (c *config) writeFile(path string) error {
	bytes, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// get current permissions and write with same permissions
	var mode os.FileMode
	info, err := os.Stat(path)
	if err != nil {
		// If the file isn't here we still want to write it
		// default to 0666
		mode = 0666
	} else {
		mode = info.Mode()
	}
	return ioutil.WriteFile(path, bytes, mode.Perm())
}

func (c *config) merge(in *config) {
	// if nil nothing to add
	if in == nil {
		return
	}
	c.Hooks.Prestart = mergeHook(c.Hooks.Prestart, in.Hooks.Prestart)
	c.Hooks.CreateRuntime = mergeHook(c.Hooks.CreateRuntime, in.Hooks.CreateRuntime)
	c.Hooks.CreateContainer = mergeHook(c.Hooks.CreateContainer, in.Hooks.CreateContainer)
	c.Hooks.StartContainer = mergeHook(c.Hooks.StartContainer, in.Hooks.StartContainer)
	c.Hooks.Poststart = mergeHook(c.Hooks.Poststart, in.Hooks.Poststart)
	c.Hooks.Poststop = mergeHook(c.Hooks.Poststop, in.Hooks.Poststop)
}

func readHooks(path string) (*config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	h := config{
		Hooks: &hooks{},
	}
	if err = json.Unmarshal(bytes, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

func mergeHook(a, b []json.RawMessage) []json.RawMessage {
	if a == nil && b == nil {
		return []json.RawMessage{}
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return append(b, a...)
}
