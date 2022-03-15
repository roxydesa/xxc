package xset

import (
	"encoding/json"
)

type XSet struct {
	CxxOutDir  string `json:"cxx_out_dir"`
	CxxOutName string `json:"cxx_out_name"`
	OutName    string `json:"out_name"`
}

// Load loads XSet from json string.
func Load(jsonbytes []byte) (*XSet, error) {
	set := XSet{
		CxxOutDir:  "./dist",
		CxxOutName: "x.cxx",
		OutName:    "main",
	}
	err := json.Unmarshal(jsonbytes, &set)
	if err != nil {
		return nil, err
	}
	return &set, nil
}