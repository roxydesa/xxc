// Copyright 2021 The X Programming Language.
// Use of this source code is governed by a BSD 3-Clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/the-xlang/x/parser"
	"github.com/the-xlang/x/pkg/io"
	"github.com/the-xlang/x/pkg/x"
	"github.com/the-xlang/x/pkg/x/xset"
)

func help(cmd string) {
	if cmd != "" {
		println("This module can only be used as single!")
		return
	}
	helpContent := [][]string{
		{"help", "Show help."},
		{"version", "Show version."},
		{"init", "Initialize new project here."},
	}
	maxlen := len(helpContent[0][0])
	for _, part := range helpContent {
		length := len(part[0])
		if length > maxlen {
			maxlen = length
		}
	}
	var sb strings.Builder
	const space = 5 // Space of between command name and description.
	for _, part := range helpContent {
		sb.WriteString(part[0])
		sb.WriteString(strings.Repeat(" ", (maxlen-len(part[0]))+space))
		sb.WriteString(part[1])
		sb.WriteByte('\n')
	}
	println(sb.String()[:sb.Len()-1])
}

func version(cmd string) {
	if cmd != "" {
		println("This module can only be used as single!")
		return
	}
	println("The X Programming Language\n" + x.Version)
}

func initProject(cmd string) {
	if cmd != "" {
		println("This module can only be used as single!")
		return
	}
	content := []byte(`{
  "cxx_out_dir": "./dist/",
  "cxx_out_name": "x.cxx",
  "out_name": "main"
}`)
	err := ioutil.WriteFile(x.SettingsFile, content, 0666)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	println("Initialized project.")
}

func processCommand(namespace, cmd string) bool {
	switch namespace {
	case "help":
		help(cmd)
	case "version":
		version(cmd)
	case "init":
		initProject(cmd)
	default:
		return false
	}
	return true
}

func init() {
	x.ExecutablePath = filepath.Dir(os.Args[0])
	// Not started with arguments.
	// Here is "2" but "os.Args" always have one element for store working directory.
	if len(os.Args) < 2 {
		os.Exit(0)
	}
	var sb strings.Builder
	for _, arg := range os.Args[1:] {
		sb.WriteString(" " + arg)
	}
	os.Args[0] = sb.String()[1:]
	arg := os.Args[0]
	index := strings.Index(arg, " ")
	if index == -1 {
		index = len(arg)
	}
	if processCommand(arg[:index], arg[index:]) {
		os.Exit(0)
	}
}

func loadXSet() {
	// File check.
	info, err := os.Stat(x.SettingsFile)
	if err != nil || info.IsDir() {
		println(`X settings file ("` + x.SettingsFile + `") is not found!`)
		os.Exit(0)
	}
	bytes, err := os.ReadFile(x.SettingsFile)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	x.XSet, err = xset.Load(bytes)
	if err != nil {
		println("X settings has errors;")
		println(err.Error())
		os.Exit(0)
	}
}

func printerr(errors []string) {
	defer os.Exit(0)
	for _, msg := range errors {
		fmt.Println(msg)
	}
}

func appendStandard(code *string) {
	year, month, day := time.Now().Date()
	hour, min, _ := time.Now().Clock()
	timeString := fmt.Sprintf("%d/%d/%d %d.%d (DD/MM/YYYY) (HH.MM)",
		day, month, year, hour, min)
	*code = `// Auto generated by X compiler.
// X compiler version: ` + x.Version + `
// Date:               ` + timeString + `

// region X_STANDARD_IMPORTS
#include <iostream>
#include <string>
#include <functional>
#include <vector>
#include <locale>
// endregion X_STANDARD_IMPORTS

// region X_CXX_API
// region X_BUILTIN_VALUES
#define nil nullptr
// endregion X_BUILTIN_VALUES

// region X_MISC
#define XALLOC(_Alloc) new(std::nothrow) _Alloc
#define XPANIC(_Msg) std::wcout << _Msg << std::endl; std::exit(EXIT_FAILURE)

template <typename _Enum_t, typename _Index_t, typename _Item_t>
static inline void foreach(const _Enum_t _Enum,
                           const std::function<void(_Index_t, _Item_t)> _Body) {
  _Index_t _index{0};
  for (auto _item: _Enum) { _Body(_index++, _item); }
}

template <typename _Enum_t, typename _Index_t>
static inline void foreach(const _Enum_t _Enum,
                           const std::function<void(_Index_t)> _Body) {
  _Index_t _index{0};
  for (auto _: _Enum) { _Body(_index++); }
}
// endregion X_MISC

// region X_BUILTIN_TYPES
typedef int8_t   i8;
typedef int16_t  i16;
typedef int32_t  i32;
typedef int64_t  i64;
typedef ssize_t  ssize;
typedef uint8_t  u8;
typedef uint16_t u16;
typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t   size;
typedef float    f32;
typedef double   f64;
typedef wchar_t  rune;

class str: public std::basic_string<rune> {
public:
// region CONSTRUCTOR
  str(void): str(L"")   { }
  str(const rune* _Str) { this->assign(_Str); }
// endregion CONSTRUCTOR

// region OPERATOR_OVERFLOWS
  rune& operator[](const ssize _Index) noexcept {
    if (_Index < 0) {
      XPANIC(L"stackoverflow exception:\n index is less than zero");
    } else if (_Index >= this->length()) {
      XPANIC(L"stackoverflow exception:\nindex overflow " +
      std::to_wstring(_Index) + L":" + std::to_wstring(this->length()));
    }
    return this->at(_Index);
  }
// endregion OPERATOR_OVERFLOWS
  };
// endregion X_BUILTIN_TYPES

// region X_STRUCTURES
template<typename _Item_t>
class array {
public:
// region FIELDS
  std::vector<_Item_t> _buffer;
// endregion FIELDS

// region CONSTRUCTORS
  array<_Item_t>(void)                                                     { this->_buffer = { }; }
  array<_Item_t>(const std::vector<_Item_t>& _Src)                         { this->_buffer = _Src; }
  array<_Item_t>(std::nullptr_t): array<_Item_t>()                         { }
  array<_Item_t>(const array<_Item_t>& _Src): array<_Item_t>(_Src._buffer) { }
// endregion CONSTRUCTORS

// region DESTRUCTOR
  ~array<_Item_t>(void) { this->_buffer.clear(); }
// endregion DESTRUCTOR

// region FOREACH_SUPPORT
  typedef _Item_t       *iterator;
  typedef const _Item_t *const_iterator;
  iterator begin(void)             { return &this->_buffer[0]; }
  const_iterator begin(void) const { return &this->_buffer[0]; }
  iterator end(void)               { return &this->_buffer[this->_buffer.size()]; }
  const_iterator end(void) const   { return &this->_buffer[this->_buffer.size()]; }
// endregion FOREACH_SUPPORT

// region OPERATOR_OVERFLOWS
  bool operator==(const array<_Item_t> &_Src) {
    const size _length = this->_buffer.size();
    const size _Src_length = _Src._buffer.size();
    if (_length != _Src_length) { return false; }
    for (size _index = 0; _index < _length; ++_index)
    { if (this->_buffer[_index] != _Src._buffer[_index]) { return false; } }
    return true;
  }

  bool operator==(std::nullptr_t)             { return this->_buffer.empty(); }
  bool operator!=(const array<_Item_t> &_Src) { return !(*this == _Src); }
  bool operator!=(std::nullptr_t)             { return !this->_buffer.empty(); }

  _Item_t& operator[](const ssize _Index) {
    const size _length = this->_buffer.size();
         if (_Index < 0) { XPANIC(L"stackoverflow exception:\n index is less than zero"); }
    else if (_Index >= _length) {
      XPANIC(L"stackoverflow exception:\nindex overflow " +
        std::to_wstring(_Index) + L":" + std::to_wstring(_length));
    }
    return this->_buffer[_Index];
  }

  friend std::wostream& operator<<(std::wostream &_Stream,
                                   const array<_Item_t> &_Src) {
    _Stream << L"[";
    const size _length = _Src._buffer.size();
    for (size _index = 0; _index < _length;) {
      _Stream << _Src._buffer[_index++];
      if (_index < _length) { _Stream << L", "; }
    }
    _Stream << L"]";
    return _Stream;
  }
// endregion OPERATOR_OVERFLOWS
};
// endregion X_STRUCTURES

// region X_BUILTIN_FUNCTIONS
template <typename _Obj_t>
static inline void _out(_Obj_t _Obj) { std::wcout << _Obj; }

template <typename _Obj_t>
static inline void _outln(_Obj_t _Obj) {
  _out<_Obj_t>(_Obj);
  std::wcout << std::endl;
}
// endregion X_BUILTIN_FUNCTIONS
// endregion X_CXX_API

// region TRANSPILED_X_CODE
` + *code + `
// endregion TRANSPILED_X_CODE

// region X_ENTRY_POINT
int main() {
// region X_ENTRY_POINT_STANDARD_CODES
  std::setlocale(LC_ALL, "");
// endregion X_ENTRY_POINT_STANDARD_CODES
  _main();

// region X_ENTRY_POINT_END_STANDARD_CODES
  return EXIT_SUCCESS;
// endregion X_ENTRY_POINT_END_STANDARD_CODES
}
// endregion X_ENTRY_POINT`
}

func writeCxxOutput(info *parser.ParseFileInfo) {
	path := filepath.Join(x.XSet.CxxOutDir, x.XSet.CxxOutName)
	err := os.MkdirAll(x.XSet.CxxOutDir, 0777)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	content := []byte(info.X_CXX)
	err = ioutil.WriteFile(path, content, 0666)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
}

var routines *sync.WaitGroup

func main() {
	f, err := io.Openfx(os.Args[0])
	if err != nil {
		println(err.Error())
		return
	}
	loadXSet()
	routines = new(sync.WaitGroup)
	info := new(parser.ParseFileInfo)
	info.File = f
	info.Routines = routines
	routines.Add(1)
	go parser.ParseFileAsync(info)
	routines.Wait()
	if info.Errors != nil {
		printerr(info.Errors)
	}
	appendStandard(&info.X_CXX)
	writeCxxOutput(info)
}
