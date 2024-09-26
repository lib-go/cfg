package go_cfg

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTempFileName(format ConfigFormat) (name string) {
	dir := os.TempDir()
	name = filepath.Join(dir, fmt.Sprintf("%v.%v", time.Now().UnixNano(), format))
	_ = os.WriteFile(name, []byte{}, 0600)
	return
}

func writeFile(name string, data string) {
	_ = os.WriteFile(name, []byte(data), 0600)
}

func readFile(name string) string {
	data, _ := os.ReadFile(name)
	return string(data)
}

func TestConfigStore_Load_map(t *testing.T) {
	model := map[string]string{}
	fn := newTempFileName(FormatYAML)
	t.Log(fn)
	writeFile(fn, "hello: world")
	cs, e := NewConfigStore(fn, &model)
	assert.Nil(t, e)
	assert.NotNil(t, cs)
	assert.Equal(t, map[string]string{"hello": "world"}, model)

	// 修改配置文件，重新载入可以使用新的值
	writeFile(fn, "hello: world2")
	_ = cs.Load()
	assert.Equal(t, map[string]string{"hello": "world2"}, model)
}

type testModel struct {
	Hello string
	List  []string
}

func TestConfigStore_Load_struct(t *testing.T) {
	model := testModel{
		"default",
		[]string{"1"},
	}

	fn := newTempFileName(FormatYAML)
	writeFile(fn, "hello: world1")
	cs, e := NewConfigStore(fn, &model)
	assert.Nil(t, e)
	assert.NotNil(t, cs)
	assert.Equal(t, "world1", model.Hello)
	//pretty.Println(model)

	// 修改配置文件，重新载入可以使用新的值

	writeFile(fn, "hello: world2")
	_ = cs.Load()
	assert.Equal(t, "world2", model.Hello)
	assert.Equal(t, []string{"1"}, model.List, "配置文件没有提供的值，不会覆盖原有值")

	// 修改配置文件，重新载入可以使用新的值
	writeFile(fn, "list: [\"1\", \"2\"]")
	_ = cs.Load()
	assert.Equal(t, "default", model.Hello, "配置文件没有提供的值，不会覆盖原有值")
	assert.Equal(t, []string{"1", "2"}, model.List, "配置文件提供的值，会覆盖原有值")
}

func TestConfigStore_Save(t *testing.T) {
	model := testModel{
		"default",
		[]string{"1"},
	}

	fn := newTempFileName(FormatYAML)
	cs, e := NewConfigStore(fn, &model)
	assert.Nil(t, e)
	_ = cs.Save()

	text := readFile(fn)
	assert.Equal(t, "hello: default\nlist:\n    - \"1\"\n", text)
	//pretty.Println(string(data))
}

func TestConfigStore_Watch(t *testing.T) {
	model := testModel{
		"default",
		[]string{"1"},
	}

	fn := newTempFileName(FormatYAML)
	cs, _ := NewConfigStore(fn, &model)

	initUpdated := false
	fileChanged := false
	ch := make(chan bool, 1)
	cs.Watch(func(changed bool) {
		if changed {
			fileChanged = true
			ch <- true
		} else {
			initUpdated = true
		}
	}, true)
	assert.True(t, initUpdated)
	assert.False(t, fileChanged)

	// 修改配置文件，重新载入可以使用新的值
	writeFile(fn, "hello: world")
	select {
	case <-ch:
		assert.True(t, fileChanged)
	case <-time.After(time.Second):
		assert.Fail(t, "timeout")
	}

	// StopWatch后，修改配置文件，就不会进入onUpdate函数
	cs.StopWatch()
	writeFile(fn, "hello: world")
	select {
	case <-ch:
		assert.Fail(t, "不应该发送事件")
	case <-time.After(time.Second * 2):
		t.Log("超时是对的")
	}
}

func TestMustBindConfig(t *testing.T) {
	model := testModel{
		"default",
		[]string{"1"},
	}

	fn := newTempFileName(FormatYAML)

	initUpdated := false
	fileChanged := false
	ch := make(chan bool, 1)

	MustBindConfig(fn, &model, func(changed bool) {
		if changed {
			fileChanged = true
			ch <- true
		} else {
			initUpdated = true
		}
	})
	assert.True(t, initUpdated)
	assert.False(t, fileChanged)

	// 修改配置文件，重新载入可以使用新的值
	writeFile(fn, "hello: world")
	select {
	case <-ch:
		assert.True(t, fileChanged)
	case <-time.After(time.Second):
		assert.Fail(t, "timeout")
	}
}
