package jsonpath

import (
	"testing"
)

func TestGetByPathSimpleDict(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": 42,
			},
		},
	}
	result, err := GetByPath("a.b.c", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0] != 42 {
		t.Fatalf("expected [42], got %v", result)
	}
}

func TestGetByPathWithDefault(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{},
		},
	}
	result, err := GetByPath("a.b.c", data, WithDefault(100))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0] != 100 {
		t.Fatalf("expected [100], got %v", result)
	}
}

func TestGetByPathSilentFalseRaises(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{},
		},
	}
	_, err := GetByPath("a.b.c.d", data, WithSilent(false))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetByPathListIndex(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"id": 1},
			map[string]any{"id": 2},
		},
	}

	result, err := GetByPath("items.0.id", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0] != 1 {
		t.Fatalf("expected [1], got %v", result)
	}

	result, err = GetByPath("items.1.id", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0] != 2 {
		t.Fatalf("expected [2], got %v", result)
	}
}

func TestGetByPathWildcard(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"id": 1},
			map[string]any{"id": 2},
		},
	}
	result, err := GetByPath("items.*.id", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0] != 1 || result[1] != 2 {
		t.Fatalf("expected [1, 2], got %v", result)
	}
}

func TestSetByPathSimpleDict(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{},
		},
	}
	err := SetByPath("a.b.c", data, 99)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{
		"a": map[string]any{
			"b": map[string]any{"c": 99},
		},
	}
	if !mapsEqual(data, expected) {
		t.Fatalf("expected %v, got %v", expected, data)
	}
}

func TestSetByPathCreateMissing(t *testing.T) {
	data := map[string]any{}
	err := SetByPath("x.y.z", data, "value")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{
		"x": map[string]any{
			"y": map[string]any{"z": "value"},
		},
	}
	if !mapsEqual(data, expected) {
		t.Fatalf("expected %v, got %v", expected, data)
	}
}

func TestSetByPathListAppend(t *testing.T) {
	data := map[string]any{
		"list": []any{},
	}
	err := SetByPath("list.!a", data, "new")
	if err != nil {
		t.Fatal(err)
	}
	list := data["list"].([]any)
	if len(list) != 1 || list[0] != "new" {
		t.Fatalf("expected [\"new\"], got %v", list)
	}
}

func TestSetByPathListIndex(t *testing.T) {
	data := map[string]any{
		"list": []any{"a", "b", "c"},
	}
	err := SetByPath("list.1", data, "B")
	if err != nil {
		t.Fatal(err)
	}
	list := data["list"].([]any)
	if len(list) != 3 || list[1] != "B" {
		t.Fatalf("expected [\"a\", \"B\", \"c\"], got %v", list)
	}
}

func TestDelByPathSimple(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{"c": 42},
		},
	}
	err := DelByPath("a.b.c", data)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{
		"a": map[string]any{
			"b": map[string]any{},
		},
	}
	if !mapsEqual(data, expected) {
		t.Fatalf("expected %v, got %v", expected, data)
	}
}

func TestDelByPathListIndex(t *testing.T) {
	data := map[string]any{
		"list": []any{"x", "y", "z"},
	}
	err := DelByPath("list.1", data)
	if err != nil {
		t.Fatal(err)
	}
	list := data["list"].([]any)
	if len(list) != 2 || list[0] != "x" || list[1] != "z" {
		t.Fatalf("expected [\"x\", \"z\"], got %v", list)
	}
}

func TestDelByPathWildcard(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"id": 1},
			map[string]any{"id": 2},
		},
	}
	err := DelByPath("items.*.id", data)
	if err != nil {
		t.Fatal(err)
	}
	items := data["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for i, item := range items {
		m := item.(map[string]any)
		if len(m) != 0 {
			t.Fatalf("item %d expected empty map, got %v", i, m)
		}
	}
}

func TestRebuildSimple(t *testing.T) {
	data := map[string]any{
		"source": map[string]any{"value": 5},
	}
	result, err := Rebuild(data, "source.value->target")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{"target": 5}
	if !mapsEqual(result.(map[string]any), expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestRebuildMultiplePaths(t *testing.T) {
	data := map[string]any{"a": 1, "b": 2}
	result, err := Rebuild(data, "a->x", "b->y")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{"x": 1, "y": 2}
	if !mapsEqual(result.(map[string]any), expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestRebuildWildcard(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"id": 10},
			map[string]any{"id": 20},
		},
	}
	result, err := Rebuild(data, "items.*.id-> *.id")
	if err != nil {
		t.Fatal(err)
	}
	list := result.([]any)
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	for i, item := range list {
		m := item.(map[string]any)
		if m["id"] != (i+1)*10 {
			t.Fatalf("item %d expected id=%d, got %v", i, (i+1)*10, m)
		}
	}
}

func TestDeepMergeBasic(t *testing.T) {
	old := map[string]any{"a": 1, "b": map[string]any{"c": 2}}
	newMap := map[string]any{"b": map[string]any{"d": 3}, "e": 4}
	merged := DeepMerge(old, newMap)
	expected := map[string]any{"a": 1, "b": map[string]any{"c": 2, "d": 3}, "e": 4}
	if !mapsEqual(merged, expected) {
		t.Fatalf("expected %v, got %v", expected, merged)
	}
	// оригинал не изменён
	if len(old) != 2 {
		t.Fatalf("original was modified: %v", old)
	}
}

func TestDeepMergeOverwrite(t *testing.T) {
	old := map[string]any{"a": 1, "b": map[string]any{"c": 2}}
	newMap := map[string]any{"b": map[string]any{"c": 99}}
	merged := DeepMerge(old, newMap)
	expected := map[string]any{"a": 1, "b": map[string]any{"c": 99}}
	if !mapsEqual(merged, expected) {
		t.Fatalf("expected %v, got %v", expected, merged)
	}
}

func TestDeepMergeNestedDict(t *testing.T) {
	old := map[string]any{"x": map[string]any{"y": map[string]any{"z": 1}}}
	newMap := map[string]any{"x": map[string]any{"y": map[string]any{"w": 2}}}
	merged := DeepMerge(old, newMap)
	expected := map[string]any{"x": map[string]any{"y": map[string]any{"z": 1, "w": 2}}}
	if !mapsEqual(merged, expected) {
		t.Fatalf("expected %v, got %v", expected, merged)
	}
}

func TestDeepMergeEmptyNew(t *testing.T) {
	old := map[string]any{"a": 1}
	newMap := map[string]any{}
	merged := DeepMerge(old, newMap)
	expected := map[string]any{"a": 1}
	if !mapsEqual(merged, expected) {
		t.Fatalf("expected %v, got %v", expected, merged)
	}
}

func TestDeepMergeEmptyOld(t *testing.T) {
	old := map[string]any{}
	newMap := map[string]any{"a": 1}
	merged := DeepMerge(old, newMap)
	expected := map[string]any{"a": 1}
	if !mapsEqual(merged, expected) {
		t.Fatalf("expected %v, got %v", expected, merged)
	}
}

func TestDeepCopyMap(t *testing.T) {
	original := map[string]any{
		"a": 1,
		"b": map[string]any{
			"c": 2,
			"d": []any{3, 4, 5},
		},
		"e": []any{
			map[string]any{"f": 6},
		},
	}

	copied := DeepCopy(original).(map[string]any)

	// Меняем копию
	copied["a"] = 99
	copied["b"].(map[string]any)["c"] = 99
	copied["b"].(map[string]any)["d"].([]any)[0] = 99
	copied["e"].([]any)[0].(map[string]any)["f"] = 99

	// Оригинал не должен измениться
	if original["a"] != 1 {
		t.Fatalf("original.a changed: expected 1, got %v", original["a"])
	}
	if original["b"].(map[string]any)["c"] != 2 {
		t.Fatalf("original.b.c changed: expected 2, got %v", original["b"].(map[string]any)["c"])
	}
	if original["b"].(map[string]any)["d"].([]any)[0] != 3 {
		t.Fatalf("original.b.d[0] changed: expected 3, got %v", original["b"].(map[string]any)["d"].([]any)[0])
	}
	if original["e"].([]any)[0].(map[string]any)["f"] != 6 {
		t.Fatalf("original.e[0].f changed: expected 6, got %v", original["e"].([]any)[0].(map[string]any)["f"])
	}
}

func TestDeepCopyList(t *testing.T) {
	original := []any{
		map[string]any{"x": 1},
		[]any{2, 3},
	}

	copied := DeepCopy(original).([]any)

	// Меняем копию
	copied[0].(map[string]any)["x"] = 99
	copied[1].([]any)[0] = 99

	// Оригинал не должен измениться
	if original[0].(map[string]any)["x"] != 1 {
		t.Fatalf("original[0].x changed: expected 1, got %v", original[0].(map[string]any)["x"])
	}
	if original[1].([]any)[0] != 2 {
		t.Fatalf("original[1][0] changed: expected 2, got %v", original[1].([]any)[0])
	}
}

func TestDeepCopyScalar(t *testing.T) {
	if DeepCopy(42) != 42 {
		t.Fatal("int deepcopy failed")
	}
	if DeepCopy("hello") != "hello" {
		t.Fatal("string deepcopy failed")
	}
	if DeepCopy(true) != true {
		t.Fatal("bool deepcopy failed")
	}
	if DeepCopy(nil) != nil {
		t.Fatal("nil deepcopy failed")
	}
}

// mapsEqual — рекурсивное сравнение map[string]any.
func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		switch ma := va.(type) {
		case map[string]any:
			mb, ok := vb.(map[string]any)
			if !ok || !mapsEqual(ma, mb) {
				return false
			}
		case []any:
			mb, ok := vb.([]any)
			if !ok || !slicesEqual(ma, mb) {
				return false
			}
		default:
			if va != vb {
				return false
			}
		}
	}
	return true
}

func slicesEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i, va := range a {
		switch ma := va.(type) {
		case map[string]any:
			mb, ok := b[i].(map[string]any)
			if !ok || !mapsEqual(ma, mb) {
				return false
			}
		case []any:
			mb, ok := b[i].([]any)
			if !ok || !slicesEqual(ma, mb) {
				return false
			}
		default:
			if va != b[i] {
				return false
			}
		}
	}
	return true
}