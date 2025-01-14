package xset

import "encoding/json"

const (
	ModeTranspile = "transpile"
	ModeCompile   = "compile"
)

type XSet struct {
	CppOutDir    string   `json:"cpp_out_dir"`
	CppOutName   string   `json:"cpp_out_name"`
	OutName      string   `json:"out_name"`
	Language     string   `json:"language"`
	Mode         string   `json:"mode"`
	PostCommands []string `json:"post_commands"`
	Indent       string   `json:"indent"`
	IndentCount  int      `json:"indent_count"`
}

// Default XSet instance.
var Default = &XSet{
	CppOutDir:    "./dist",
	CppOutName:   "x.cpp",
	OutName:      "main",
	Language:     "",
	Mode:         "transpile",
	Indent:       "\t",
	IndentCount:  1,
	PostCommands: []string{},
}

// Load loads XSet from json string.
func Load(bytes []byte) (*XSet, error) {
	set := *Default
	err := json.Unmarshal(bytes, &set)
	if err != nil {
		return nil, err
	}
	return &set, nil
}
