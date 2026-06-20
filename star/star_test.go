package star

import (
	"strings"
	"testing"

	"go.starlark.net/starlark"
)

// ========================================
// Result tests
// ========================================

func TestResult_InitialState(t *testing.T) {
	res := &Result{}
	if res.Success {
		t.Error("expected Success=false")
	}
	if res.Prints != nil {
		t.Error("expected Prints=nil")
	}
	if res.Error != nil {
		t.Error("expected Error=nil")
	}
	if res.Value != nil {
		t.Error("expected Value=nil")
	}
}

// ========================================
// Runner tests
// ========================================

func TestRunner_Init(t *testing.T) {
	r := NewRunner()
	if r == nil {
		t.Fatal("NewRunner() returned nil")
	}
	if r.setup != DefaultSetup {
		t.Error("expected default setup")
	}
	if r.wrapper != DefaultWrapper {
		t.Error("expected default wrapper")
	}
}

func TestRunner_WithSetup(t *testing.T) {
	customSetup := Setup(`def foo(): return 42`)
	r := NewRunner(WithSetup(customSetup))
	if r.setup != customSetup {
		t.Error("expected custom setup")
	}
}

func TestRunner_WithWrapper(t *testing.T) {
	customWrapper := Wrapper(`{setup}
def process(input):
{script}
def main(inp):
    result = process(inp)
    return {'custom': result}
results = main(input)
`)
	r := NewRunner(WithWrapper(customWrapper))
	if r.wrapper != customWrapper {
		t.Error("expected custom wrapper")
	}
}

func TestRunner_Run_Basic(t *testing.T) {
	r := NewRunner()
	script := "return input['x']"

	dict := starlark.NewDict(1)
	dict.SetKey(starlark.String("x"), starlark.MakeInt(42))

	res := r.Run(script, dict)
	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}
}

func TestRunner_Run_ReturnScalar(t *testing.T) {
	r := NewRunner()
	script := "return 42"
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}
	if res.Value == nil {
		t.Fatal("expected non-nil result")
	}

	// Должен быть обёрнут в {'result': 42}
	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	val, found, err := dict.Get(starlark.String("result"))
	if !found || err != nil {
		t.Fatal("expected key 'result'")
	}
	if val != starlark.MakeInt(42) {
		t.Errorf("expected 42, got %v", val)
	}
}

func TestRunner_Run_ReturnDict(t *testing.T) {
	r := NewRunner()
	script := "return {'answer': 42}"
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	val, found, err := dict.Get(starlark.String("answer"))
	if !found || err != nil {
		t.Fatal("expected key 'answer'")
	}
	if val != starlark.MakeInt(42) {
		t.Errorf("expected 42, got %v", val)
	}
}

func TestRunner_Run_ReturnList(t *testing.T) {
	r := NewRunner()
	script := "return [1, 2, 3]"
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	_, ok := res.Value.(*starlark.List)
	if !ok {
		t.Fatalf("expected *starlark.List, got %T", res.Value)
	}
}

func TestRunner_Run_ReturnNone(t *testing.T) {
	r := NewRunner()
	script := "return None"
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	// None должен превратиться в пустой dict
	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	if dict.Len() != 0 {
		t.Errorf("expected empty dict, got %v", dict)
	}
}

func TestRunner_Run_WithPrint(t *testing.T) {
	r := NewRunner()
	script := `
print("hello")
print("world")
return input
`
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	if len(res.Prints) != 2 {
		t.Fatalf("expected 2 prints, got %d: %v", len(res.Prints), res.Prints)
	}
	if res.Prints[0] != "hello" {
		t.Errorf("expected 'hello', got %q", res.Prints[0])
	}
	if res.Prints[1] != "world" {
		t.Errorf("expected 'world', got %q", res.Prints[1])
	}
}

func TestRunner_Run_Error(t *testing.T) {
	r := NewRunner()
	script := "1/0"
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Error == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestRunner_Run_WithRegex(t *testing.T) {
	r := NewRunner()
	script := `
match = re.search(r'\d+', 'abc123def')
return {'match': match}
`
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	val, found, err := dict.Get(starlark.String("match"))
	if !found || err != nil {
		t.Fatal("expected key 'match'")
	}
	if string(val.(starlark.String)) != "123" {
		t.Errorf("expected '123', got %v", val)
	}
}

func TestRunner_Run_WithTime(t *testing.T) {
	r := NewRunner()
	script := `
t = time.now()
return {'time': t}
`
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	val, found, err := dict.Get(starlark.String("time"))
	if !found || err != nil {
		t.Fatal("expected key 'time'")
	}
	_, ok = val.(starlark.Float)
	if !ok {
		t.Errorf("expected Float, got %T", val)
	}
}

func TestRunner_Run_WithInput(t *testing.T) {
	r := NewRunner()
	script := "return input['x'] * 2"

	dict := starlark.NewDict(1)
	dict.SetKey(starlark.String("x"), starlark.MakeInt(21))
	data := dict

	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	val, found, err := dict.Get(starlark.String("result"))
	if !found || err != nil {
		t.Fatal("expected key 'result'")
	}
	if val != starlark.MakeInt(42) {
		t.Errorf("expected 42, got %v", val)
	}
}

func TestRunner_WrapScript_Cache(t *testing.T) {
	r := NewRunner()
	script := "return 1"

	wrapped1 := r.wrapScript(script)
	wrapped2 := r.wrapScript(script)

	if wrapped1 != wrapped2 {
		t.Error("expected cached result")
	}
}

func TestRunner_Indent(t *testing.T) {
	input := "line1\nline2\nline3"
	expected := "   line1\n   line2\n   line3"
	got := indent(input, "   ")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRunner_Indent_Empty(t *testing.T) {
	got := indent("", "   ")
	if got != "   " {
		t.Errorf("expected '   ', got %q", got)
	}
}

// ========================================
// reSearch tests
// ========================================

func TestReSearch_Found(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	fn := starlark.NewBuiltin("star_re_search", reSearch)
	args := starlark.Tuple{
		starlark.String(`\d+`),
		starlark.String("abc123def"),
	}
	val, err := fn.CallInternal(thread, args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val.(starlark.String)) != "123" {
		t.Errorf("expected '123', got %v", val)
	}
}

func TestReSearch_NotFound(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	fn := starlark.NewBuiltin("star_re_search", reSearch)
	args := starlark.Tuple{
		starlark.String(`\d+`),
		starlark.String("abcdef"),
	}
	val, err := fn.CallInternal(thread, args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != starlark.None {
		t.Errorf("expected None, got %v", val)
	}
}

func TestReSearch_WithGroup(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	fn := starlark.NewBuiltin("star_re_search", reSearch)
	args := starlark.Tuple{
		starlark.String(`(\d+)(\w+)`),
		starlark.String("123abc"),
		starlark.MakeInt(1),
	}
	val, err := fn.CallInternal(thread, args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val.(starlark.String)) != "123" {
		t.Errorf("expected '123', got %v", val)
	}
}

// ========================================
// reFindall tests
// ========================================

func TestReFindall(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	fn := starlark.NewBuiltin("star_re_findall", reFindall)
	args := starlark.Tuple{
		starlark.String(`\d+`),
		starlark.String("abc123def456ghi"),
	}
	val, err := fn.CallInternal(thread, args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(*starlark.List)
	if !ok {
		t.Fatalf("expected *starlark.List, got %T", val)
	}
	if list.Len() != 2 {
		t.Fatalf("expected 2 matches, got %d", list.Len())
	}
	if string(list.Index(0).(starlark.String)) != "123" {
		t.Errorf("expected '123', got %v", list.Index(0))
	}
	if string(list.Index(1).(starlark.String)) != "456" {
		t.Errorf("expected '456', got %v", list.Index(1))
	}
}

// ========================================
// struct builtin tests
// ========================================

func TestStructBuiltin(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	fn := starlark.NewBuiltin("struct", structBuiltin)
	kwargs := []starlark.Tuple{
		{starlark.String("foo"), starlark.MakeInt(42)},
		{starlark.String("bar"), starlark.String("baz")},
	}
	val, err := fn.CallInternal(thread, starlark.Tuple{}, kwargs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// starlarkstruct форматирует с пробелами: struct(bar = "baz", foo = 42)
	// Порядок полей не гарантирован, проверяем через contains
	s := val.String()
	if !strings.Contains(s, "foo = 42") {
		t.Errorf("expected 'foo = 42' in struct, got: %s", s)
	}
	if !strings.Contains(s, `bar = "baz"`) {
		t.Errorf("expected 'bar = \"baz\"' in struct, got: %s", s)
	}
}

// ========================================
// Custom wrapper tests
// ========================================

func TestRunner_CustomWrapper(t *testing.T) {
	customWrapper := Wrapper(`
{setup}

def process(input):
{script}

def main(inp):
    result = process(inp)
    return {'custom': result}

results = main(input)
`)
	r := NewRunner(WithWrapper(customWrapper))
	script := "return input * 2"
	data := starlark.MakeInt(5)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	val, found, err := dict.Get(starlark.String("custom"))
	if !found || err != nil {
		t.Fatal("expected key 'custom'")
	}
	if val != starlark.MakeInt(10) {
		t.Errorf("expected 10, got %v", val)
	}
}

func TestRunner_CustomSetup(t *testing.T) {
	customSetup := Setup(`
def test():
    return 42
`)
	r := NewRunner(WithSetup(customSetup))
	script := `return {"result": test()}`
	data := starlark.NewDict(0)
	res := r.Run(script, data)

	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	dict, ok := res.Value.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", res.Value)
	}
	val, found, err := dict.Get(starlark.String("result"))
	if !found || err != nil {
		t.Fatal("expected key 'result'")
	}
	if val != starlark.MakeInt(42) {
		t.Errorf("expected 42, got %v", val)
	}
}

// ========================================
// Regexp cache tests
// ========================================

func TestRegexpCache(t *testing.T) {
	cache := newRegexpCache()

	re1 := cache.get(`\d+`)
	re2 := cache.get(`\d+`)
	if re1 != re2 {
		t.Error("expected cached regexp")
	}

	re3 := cache.get(`[a-z]+`)
	if re3 == re1 {
		t.Error("expected different regexp for different pattern")
	}
}