// Copyright 2021 The X Programming Language.
// Use of this source code is governed by a BSD 3-Clause
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/the-xlang/xxc/documenter"
	"github.com/the-xlang/xxc/parser"
	"github.com/the-xlang/xxc/pkg/x"
	"github.com/the-xlang/xxc/pkg/xio"
	"github.com/the-xlang/xxc/pkg/xlog"
	"github.com/the-xlang/xxc/pkg/xset"
)

type Parser = parser.Parser

func help(cmd string) {
	if cmd != "" {
		println("This module can only be used as single!")
		return
	}
	helpmap := [][]string{
		{"help", "Show help."},
		{"version", "Show version."},
		{"init", "Initialize new project here."},
		{"doc", "Documentize X source code."},
	}
	max := len(helpmap[0][0])
	for _, key := range helpmap {
		len := len(key[0])
		if len > max {
			max = len
		}
	}
	var sb strings.Builder
	const space = 5 // Space of between command name and description.
	for _, part := range helpmap {
		sb.WriteString(part[0])
		sb.WriteString(strings.Repeat(" ", (max-len(part[0]))+space))
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
	bytes, err := json.MarshalIndent(*xset.Default, "", "  ")
	if err != nil {
		println(err)
		os.Exit(0)
	}
	err = ioutil.WriteFile(x.SettingsFile, bytes, 0666)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	println("Initialized project.")
}

func doc(cmd string) {
	cmd = strings.TrimSpace(cmd)
	paths := strings.SplitN(cmd, " ", -1)
	for _, path := range paths {
		path = strings.TrimSpace(path)
		p := compile(path, false, true)
		if p == nil {
			continue
		}
		if printlogs(p) {
			fmt.Println(x.GetErr("doc_couldnt_generated", path))
			continue
		}
		docjson, err := documenter.Documentize(p)
		if err != nil {
			fmt.Println(x.GetErr("error", err.Error()))
			continue
		}
		path = filepath.Join(x.Set.CxxOutDir, path+x.DocExt)
		writeOutput(path, docjson)
	}
}

func processCommand(namespace, cmd string) bool {
	switch namespace {
	case "help":
		help(cmd)
	case "version":
		version(cmd)
	case "init":
		initProject(cmd)
	case "doc":
		doc(cmd)
	default:
		return false
	}
	return true
}

func init() {
	execp, err := os.Executable()
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	execp = filepath.Dir(execp)
	x.ExecPath = execp
	x.StdlibPath = filepath.Join(x.ExecPath, x.Stdlib)
	x.LangsPath = filepath.Join(x.ExecPath, x.Langs)

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
	i := strings.Index(arg, " ")
	if i == -1 {
		i = len(arg)
	}
	if processCommand(arg[:i], arg[i:]) {
		os.Exit(0)
	}
}

func loadLangWarns(path string, infos []fs.FileInfo) {
	i := -1
	for j, f := range infos {
		if f.IsDir() || f.Name() != "warns.json" {
			continue
		}
		i = j
		path = filepath.Join(path, f.Name())
		break
	}
	if i == -1 {
		return
	}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		println("Language's warnings couldn't loaded (uses default);")
		println(err.Error())
		return
	}
	err = json.Unmarshal(bytes, &x.Warns)
	if err != nil {
		println("Language's warnings couldn't loaded (uses default);")
		println(err.Error())
		return
	}
}

func loadLangErrs(path string, infos []fs.FileInfo) {
	i := -1
	for j, f := range infos {
		if f.IsDir() || f.Name() != "errs.json" {
			continue
		}
		i = j
		path = filepath.Join(path, f.Name())
		break
	}
	if i == -1 {
		return
	}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		println("Language's errors couldn't loaded (uses default);")
		println(err.Error())
		return
	}
	err = json.Unmarshal(bytes, &x.Errs)
	if err != nil {
		println("Language's errors couldn't loaded (uses default);")
		println(err.Error())
		return
	}
}

func loadLang() {
	lang := strings.TrimSpace(x.Set.Language)
	if lang == "" || lang == "default" {
		return
	}
	path := filepath.Join(x.LangsPath, lang)
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		println("Language couldn't loaded (uses default);")
		println(err.Error())
		return
	}
	loadLangWarns(path, infos)
	loadLangErrs(path, infos)
}

func checkMode() {
	lower := strings.ToLower(x.Set.Mode)
	if lower != xset.ModeTranspile &&
		lower != xset.ModeCompile {
		key, _ := reflect.TypeOf(x.Set).Elem().FieldByName("Mode")
		tag := string(key.Tag)
		// 6 for skip "json:
		tag = tag[6 : len(tag)-1]
		println(x.GetErr("invalid_value_for_key", x.Set.Mode, tag))
		os.Exit(0)
	}
	x.Set.Mode = lower
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
	x.Set, err = xset.Load(bytes)
	if err != nil {
		println("X settings has errors;")
		println(err.Error())
		os.Exit(0)
	}
	loadLang()
	checkMode()
}

// printlogs prints logs and returns true
// if logs has error, false if not.
func printlogs(p *Parser) bool {
	var str strings.Builder
	for _, log := range p.Warns {
		switch log.Type {
		case xlog.FlatWarn:
			str.WriteString("WARNING: ")
			str.WriteString(log.Msg)
		case xlog.Warn:
			str.WriteString("WARNING: ")
			str.WriteString(log.Path)
			str.WriteByte(':')
			str.WriteString(fmt.Sprint(log.Row))
			str.WriteByte(':')
			str.WriteString(fmt.Sprint(log.Column))
			str.WriteByte(' ')
			str.WriteString(log.Msg)
		}
		str.WriteByte('\n')
	}
	for _, log := range p.Errs {
		switch log.Type {
		case xlog.FlatErr:
			str.WriteString("ERROR: ")
			str.WriteString(log.Msg)
		case xlog.Err:
			str.WriteString("ERROR: ")
			str.WriteString(log.Path)
			str.WriteByte(':')
			str.WriteString(fmt.Sprint(log.Row))
			str.WriteByte(':')
			str.WriteString(fmt.Sprint(log.Column))
			str.WriteByte(' ')
			str.WriteString(log.Msg)
		}
		str.WriteByte('\n')
	}
	print(str.String())
	return len(p.Errs) > 0
}

func appendStandard(code *string) {
	year, month, day := time.Now().Date()
	hour, min, _ := time.Now().Clock()
	timeStr := fmt.Sprintf("%d/%d/%d %d.%d (DD/MM/YYYY) (HH.MM)",
		day, month, year, hour, min)
	*code = `// Auto generated by XXC compiler.
// X compiler version: ` + x.Version + `
// Date:               ` + timeStr + `

#if defined(WIN32) || defined(_WIN32) || defined(__WIN32__) || defined(__NT__)
#define _WINDOWS
#endif

// region X_STANDARD_IMPORTS
#include <iostream>
#include <string>
#include <string.h>
#include <functional>
#include <vector>
#include <map>
// endregion X_STANDARD_IMPORTS

// region X_CXX_API
// region X_BUILTIN_VALUES
#define nil nullptr
// endregion X_BUILTIN_VALUES

// region X_BUILTIN_TYPES
typedef int8_t        i8;
typedef int16_t       i16;
typedef int32_t       i32;
typedef int64_t       i64;
typedef uint8_t       u8;
typedef uint16_t      u16;
typedef uint32_t      u32;
typedef uint64_t      u64;
typedef std::size_t   size;
typedef float         f32;
typedef double        f64;
typedef void          *voidptr;
#define func          std::function

// region X_STRUCTURES
template<typename _Item_t>
class array {
public:
  std::vector<_Item_t> _buffer{};

  array<_Item_t>(void) noexcept                       {}
  array<_Item_t>(const std::nullptr_t) noexcept       {}
  array<_Item_t>(const array<_Item_t>& _Src) noexcept { this->_buffer = _Src._buffer; }

  array<_Item_t>(const std::initializer_list<_Item_t> &_Src) noexcept
  { this->_buffer = std::vector<_Item_t>(_Src.begin(), _Src.end()); }

  ~array<_Item_t>(void) noexcept { this->_buffer.clear(); }

  typedef _Item_t       *iterator;
  typedef const _Item_t *const_iterator;
  iterator begin(void) noexcept             { return &this->_buffer[0]; }
  const_iterator begin(void) const noexcept { return &this->_buffer[0]; }
  iterator end(void) noexcept               { return &this->_buffer[this->_buffer.size()]; }
  const_iterator end(void) const noexcept   { return &this->_buffer[this->_buffer.size()]; }

  inline void clear(void) noexcept { this->_buffer.clear(); }
  inline size len(void) const noexcept { return this->_buffer.size(); }

  _Item_t *find(const _Item_t &_Item) noexcept {
    iterator _it{this->begin()};
    const iterator _end{this->end()};
    for (; _it < _end; ++_it)
    { if (_Item == *_it) { return _it; } }
    return nil;
  }

  _Item_t *rfind(const _Item_t &_Item) noexcept {
    iterator _it{this->end()};
    const iterator _begin{this->begin()};
    for (; _it >= _begin; --_it)
    { if (_Item == *_it) { return _it; } }
    return nil;
  }

  void erase(const _Item_t &_Item) noexcept {
    auto _it{this->_buffer.begin()};
    auto _end{this->_buffer.end()};
    for (; _it < _end; ++_it) {
      if (_Item == *_it) {
        this->_buffer.erase(_it);
        return;
      }
    }
  }

  void erase_all(const _Item_t &_Item) noexcept {
    auto _it{this->_buffer.begin()};
    auto _end{this->_buffer.end()};
    for (; _it < _end; ++_it)
    { if (_Item == *_it) { this->_buffer.erase(_it); } }
  }

  void append(const array<_Item_t> &_Items) noexcept {
    for (const _Item_t _item: _Items) { this->_buffer.push_back(_item); }
  }

  bool insert(const size &_Start, const array<_Item_t> &_Items) noexcept {
    auto _it{this->_buffer.begin()+_Start};
    if (_it >= this->_buffer.end()) { return false; }
    this->_buffer.insert(_it, _Items.begin(), _Items.end());
    return true;
  }

  inline bool empty(void) const noexcept { return this->_buffer.empty(); }

  bool operator==(const array<_Item_t> &_Src) const noexcept {
    const size _length = this->_buffer.size();
    const size _Src_length = _Src._buffer.size();
    if (_length != _Src_length) { return false; }
    for (size _index = 0; _index < _length; ++_index)
    { if (this->_buffer[_index] != _Src._buffer[_index]) { return false; } }
    return true;
  }

  bool operator==(const std::nullptr_t) const noexcept       { return this->_buffer.empty(); }
  bool operator!=(const array<_Item_t> &_Src) const noexcept { return !(*this == _Src); }
  bool operator!=(const std::nullptr_t) const noexcept       { return !this->_buffer.empty(); }
  _Item_t& operator[](const size _Index)                     { return this->_buffer[_Index]; }

  friend std::ostream& operator<<(std::ostream &_Stream,
                                  const array<_Item_t> &_Src) {
    _Stream << '[';
    const size _length = _Src._buffer.size();
    for (size _index = 0; _index < _length;) {
      _Stream << _Src._buffer[_index++];
      if (_index < _length) { _Stream << u8", "; }
    }
    _Stream << ']';
    return _Stream;
  }
};

template<typename _Key_t, typename _Value_t>
class map: public std::map<_Key_t, _Value_t> {
public:
  map<_Key_t, _Value_t>(void) noexcept                 {}
  map<_Key_t, _Value_t>(const std::nullptr_t) noexcept {}
  map<_Key_t, _Value_t>(const std::initializer_list<std::pair<_Key_t, _Value_t>> _Src)
  { for (const auto _data: _Src) { this->insert(_data); } }

  array<_Key_t> keys(void) const noexcept {
    array<_Key_t> _keys{};
    for (const auto _pair: *this)
    { _keys._buffer.push_back(_pair.first); }
    return _keys;
  }

  array<_Value_t> values(void) const noexcept {
    array<_Value_t> _values{};
    for (const auto _pair: *this)
    { _values._buffer.push_back(_pair.second); }
    return _values;
  }

  inline bool has(const _Key_t _Key) const noexcept { return this->find(_Key) != this->end(); }
  inline void del(const _Key_t _Key) noexcept { this->erase(_Key); }

  bool operator==(const std::nullptr_t) const noexcept { return this->empty(); }
  bool operator!=(const std::nullptr_t) const noexcept { return !this->empty(); }

  friend std::ostream& operator<<(std::ostream &_Stream,
                                  const map<_Key_t, _Value_t> &_Src) {
    _Stream << '{';
    size _length = _Src.size();
    for (const auto _pair: _Src) {
      _Stream << _pair.first;
      _Stream << ':';
      _Stream << _pair.second;
      if (--_length > 0) { _Stream << u8", "; }
    }
    _Stream << '}';
    return _Stream;
  }
};
// endregion X_STRUCTURES

class str {
public:
  std::string _buffer{};

  str(void) noexcept                   {}
  str(const char *_Src) noexcept       { this->_buffer = _Src ? _Src : ""; }
  str(const std::string _Src) noexcept { this->_buffer = _Src; }
  str(const str &_Src) noexcept        { this->_buffer = _Src._buffer; }
  
  str(const array<char> &_Src) noexcept
  { this->_buffer = std::string{_Src.begin(), _Src.end()}; }

  str(const array<u8> &_Src) noexcept
  { this->_buffer = std::string{_Src.begin(), _Src.end()}; }

  typedef char       *iterator;
  typedef const char *const_iterator;
  iterator begin(void) noexcept             { return &this->_buffer[0]; }
  const_iterator begin(void) const noexcept { return &this->_buffer[0]; }
  iterator end(void) noexcept               { return &this->_buffer[this->len()]; }
  const_iterator end(void) const noexcept   { return &this->_buffer[this->len()]; }

  inline size len(void) const noexcept { return this->_buffer.length(); }
  inline bool empty(void) const noexcept { return this->_buffer.empty(); }

  inline str sub(const size start, const size end) const noexcept
  { return this->_buffer.substr(start, end); }

  inline str sub(const size start) const noexcept
  { return this->_buffer.substr(start); }

  inline bool has_prefix(const str &_Sub) const noexcept
  { return this->len() >= _Sub.len() && this->sub(0, _Sub.len()) == _Sub._buffer; }

  inline bool has_suffix(const str &_Sub) const noexcept
  { return this->len() >= _Sub.len() && this->sub(this->len()-_Sub.len()) == _Sub; }

  inline size find(const str &_Sub) const noexcept
  { return this->_buffer.find(_Sub._buffer); }

  inline size rfind(const str &_Sub) const noexcept
  { return this->_buffer.rfind(_Sub._buffer); }

  inline const char* cstr(void) const noexcept
  { return this->_buffer.c_str(); }

  str trim(const str &_Bytes) const noexcept {
    const_iterator _it{this->begin()};
    const const_iterator _end{this->end()};
    const_iterator _begin{this->begin()};
    for (; _it < _end; ++_it) {
      bool exist{false};
      const_iterator _bytes_it{_Bytes.begin()};
      const const_iterator _bytes_end{_Bytes.end()};
      for (; _bytes_it < _bytes_end; ++_bytes_it)
      { if ((exist = *_it == *_bytes_it)) { break; } }
      if (!exist) { return this->sub(_it-_begin); }
    }
    return str{u8""};
  }

  str rtrim(const str &_Bytes) const noexcept {
    const_iterator _it{this->end()-1};
    const const_iterator _begin{this->begin()};
    for (; _it >= _begin; --_it) {
      bool exist{false};
      const_iterator _bytes_it{_Bytes.begin()};
      const const_iterator _bytes_end{_Bytes.end()};
      for (; _bytes_it < _bytes_end; ++_bytes_it)
      { if ((exist = *_it == *_bytes_it)) { break; } }
      if (!exist) { return this->sub(0, _it-_begin+1); }
    }
    return str{u8""};
  }

  array<str> split(const str &_Sub, const i64 &_N) const noexcept {
    array<str> _parts{};
    if (_N == 0) { return _parts; }
    const const_iterator _begin{this->begin()};
    std::string _s{this->_buffer};
    size _pos{std::string::npos};
    if (_N < 0) {
      while ((_pos = _s.find(_Sub._buffer)) != std::string::npos) {
        _parts._buffer.push_back(_s.substr(0, _pos));
        _s = _s.substr(_pos+_Sub.len());
      }
      if (!_parts.empty()) { _parts._buffer.push_back(str{_s}); }
    } else {
      size _n{0};
      while ((_pos = _s.find(_Sub._buffer)) != std::string::npos) {
        _parts._buffer.push_back(_s.substr(0, _pos));
        _s = _s.substr(_pos+_Sub.len());
        if (++_n >= _N) { break; }
      }
      if (!_parts.empty() && _n < _N) { _parts._buffer.push_back(str{_s}); }
    }
    return _parts;
  }

  str replace(const str &_Sub, const str &_New, const i64 &_N) const noexcept {
    if (_N == 0) { return *this; }
    std::string _s{this->_buffer};
    size start_pos{0};
    if (_N < 0) {
      while((start_pos = _s.find(_Sub._buffer, start_pos)) != std::string::npos) {
        _s.replace(start_pos, _Sub.len(), _New._buffer);
        start_pos += _New.len();
      }
    } else {
      size _n{0};
      while((start_pos = _s.find(_Sub._buffer, start_pos)) != std::string::npos) {
        _s.replace(start_pos, _Sub.len(), _New._buffer);
        start_pos += _New.len();
        if (++_n >= _N) { break; }
      }
    }
    return str{_s};
  }

  operator array<char>(void) const noexcept {
    array<char> _array{};
    _array._buffer = std::vector<char>{this->begin(), this->end()};
    return _array;
  }

  operator array<u8>(void) const noexcept {
    array<u8> _array{};
    _array._buffer = std::vector<u8>{this->begin(), this->end()};
    return _array;
  }

  operator const char*(void) { this->_buffer.c_str(); }
  operator char*(void)       { this->_buffer.c_str(); }

  char &operator[](size _Index) { return this->_buffer[_Index]; }

  void operator+=(const str _Str) noexcept        { this->_buffer += _Str._buffer; }
  str operator+(const str _Str) const noexcept    { return str{this->_buffer + _Str._buffer}; }
  bool operator==(const str &_Str) const noexcept { return this->_buffer == _Str._buffer; }
  bool operator!=(const str &_Str) const noexcept { return this->_buffer != _Str._buffer; }

  friend std::ostream& operator<<(std::ostream &_Stream, const str &_Src)
  { return _Stream << _Src._buffer; }
};
// endregion X_BUILTIN_TYPES

// region X_MISC
class exception: public std::exception {
private:
  std::basic_string<char> _buffer;
public:
  exception(const char *_Str)      { this->_buffer = _Str; }
  const char *what() const throw() { return this->_buffer.c_str(); }
};

template<typename _Alloc_t>
static inline _Alloc_t *xalloc() { return new(std::nothrow) _Alloc_t; }

template <typename _Enum_t, typename _Index_t, typename _Item_t>
static inline void foreach(const _Enum_t _Enum,
                           const func<void(_Index_t, _Item_t)> _Body) {
  _Index_t _index{0};
  for (auto _item: _Enum) { _Body(_index++, _item); }
}

template <typename _Enum_t, typename _Index_t>
static inline void foreach(const _Enum_t _Enum,
                           const func<void(_Index_t)> _Body) {
  _Index_t _index{0};
  for (auto begin = _Enum.begin(), end = _Enum.end(); begin < end; ++begin)
  { _Body(_index++); }
}

template <typename _Key_t, typename _Value_t>
static inline void foreach(const map<_Key_t, _Value_t> _Map,
                           const func<void(_Key_t)> _Body) {
  for (const auto _pair: _Map) { _Body(_pair.first); }
}

template <typename _Key_t, typename _Value_t>
static inline void foreach(const map<_Key_t, _Value_t> _Map,
                           const func<void(_Key_t, _Value_t)> _Body) {
  for (const auto _pair: _Map) { _Body(_pair.first, _pair.second); }
}

template<typename Type, unsigned N, unsigned Last>
struct tuple_ostream {
  static void arrow(std::ostream &_Stream, const Type &_Type) {
    _Stream << std::get<N>(_Type) << u8", ";
    tuple_ostream<Type, N + 1, Last>::arrow(_Stream, _Type);
  }
};

template<typename Type, unsigned N>
struct tuple_ostream<Type, N, N> {
  static void arrow(std::ostream &_Stream, const Type &_Type)
  { _Stream << std::get<N>(_Type); }
};

template<typename... Types>
std::ostream& operator<<(std::ostream &_Stream,
                         const std::tuple<Types...> &_Tuple) {
  _Stream << u8"(";
  tuple_ostream<std::tuple<Types...>, 0, sizeof...(Types)-1>::arrow(_Stream, _Tuple);
  _Stream << u8")";
  return _Stream;
}

template<typename _Function_t, typename _Tuple_t, size_t ... _I_t>
inline auto tuple_as_args(const _Function_t _Function,
                          const _Tuple_t _Tuple,
                          const std::index_sequence<_I_t ...>)
{ return _Function(std::get<_I_t>(_Tuple) ...); }

template<typename _Function_t, typename _Tuple_t>
inline auto tuple_as_args(const _Function_t _Function, const _Tuple_t _Tuple) {
  static constexpr auto _size{std::tuple_size<_Tuple_t>::value};
  return tuple_as_args(_Function, _Tuple, std::make_index_sequence<_size>{});
}

struct defer {
  typedef func<void(void)> _Function_t;
  template<class Callable>
  defer(Callable &&_function): _function(std::forward<Callable>(_function)) {}
  defer(defer &&_Src): _function(std::move(_Src._function))                 { _Src._function = nullptr; }
  ~defer() noexcept                                                         { if (this->_function) { this->_function(); } }
  defer(const defer &)          = delete;
  void operator=(const defer &) = delete;
  _Function_t _function;
};

std::ostream &operator<<(std::ostream &_Stream, const i8 &_Src)
{ return _Stream << (i32)(_Src); }

std::ostream &operator<<(std::ostream &_Stream, const u8 &_Src)
{ return _Stream << (i32)(_Src); }

#define XTHROW(_Msg) throw exception(_Msg)
#define _CONCAT(_A, _B) _A ## _B
#define CONCAT(_A, _B) _CONCAT(_A, _B)
#define DEFER(_Expr) defer CONCAT(XXDEFER_, __LINE__){[&](void) mutable -> void { _Expr; }}
#define XID(_Identifier) CONCAT(_, _Identifier)
// endregion X_MISC

// region X_BUILTIN_FUNCTIONS
template <typename _Obj_t>
static inline void XID(out)(const _Obj_t _Obj) noexcept { std::cout << _Obj; }

template <typename _Obj_t>
static inline void XID(outln)(const _Obj_t _Obj) noexcept {
  XID(out)<_Obj_t>(_Obj);
  std::cout << std::endl;
}
// endregion X_BUILTIN_FUNCTIONS
// endregion X_CXX_API

// region TRANSPILED_X_CODE
` + *code + `
// endregion TRANSPILED_X_CODE

// region X_ENTRY_POINT
int main() {
  _main();
  return EXIT_SUCCESS;
}
// endregion X_ENTRY_POINT`
}

func writeOutput(path, content string) {
	err := os.MkdirAll(x.Set.CxxOutDir, 0777)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	bytes := []byte(content)
	err = ioutil.WriteFile(path, bytes, 0666)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
}

func compile(path string, main, justDefs bool) *Parser {
	loadXSet()
	p := parser.New(nil)
	// Check standard library.
	inf, err := os.Stat(x.StdlibPath)
	if err != nil || !inf.IsDir() {
		p.Errs = append(p.Errs, xlog.CompilerLog{
			Type: xlog.FlatErr,
			Msg:  "standard library directory not found",
		})
		return p
	}

	f, err := xio.Openfx(path)
	if err != nil {
		println(err.Error())
		return nil
	}
	p.File = f
	p.Parsef(true, false)
	return p
}

func execPostCommands() {
	for _, cmd := range x.Set.PostCommands {
		fmt.Println(">", cmd)
		parts := strings.SplitN(cmd, " ", -1)
		err := exec.Command(parts[0], parts[1:]...).Run()
		if err != nil {
			println(err.Error())
		}
	}
}

func doSpell(path, cxx string) {
	defer execPostCommands()
	writeOutput(path, cxx)
	switch x.Set.Mode {
	case xset.ModeCompile:
		defer os.Remove(path)
		println("compilation is not supported yet")
	}
}

func main() {
	fpath := os.Args[0]
	p := compile(fpath, true, false)
	if p == nil {
		return
	}
	if printlogs(p) {
		os.Exit(0)
	}
	cxx := p.Cxx()
	appendStandard(&cxx)
	path := filepath.Join(x.Set.CxxOutDir, x.Set.CxxOutName)
	doSpell(path, cxx)
}
