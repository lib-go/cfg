package cfg

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTmpFile(suffix string) (f *os.File) {
	dir := os.TempDir()
	name := filepath.Join(dir, fmt.Sprintf("%v.%v", time.Now().UnixNano(), suffix))
	f, _ = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	return
}

func TestNonExistConfigFile(t *testing.T) {
	filePath := fmt.Sprintf("/tmp/config_test.json")
	if _, err := os.Stat(filePath); os.IsExist(err) {
		os.Remove(filePath)
	}

	model := map[string]string{}
	cfg, e := NewConfigStore(filePath, model)

	if e == nil {
		t.Error("should not accept non-ptr model")
	}

	cfg, e = NewConfigStore(filePath, &model)
	if e != nil {
		t.Error(e)
	}
	t.Log(cfg, e)
}

func TestEmptyConfigFile(t *testing.T) {
	f := newTmpFile("config.json")
	defer f.Close()
	t.Log("file", f.Name())

	model := map[string]string{}
	cfg, e := NewConfigStore(f.Name(), &model)
	t.Log(cfg, e)
}

func TestPartialUpdate(t *testing.T) {
	model := struct {
		Hello string
		List  []string
	}{
		"world",
		[]string{"1"},
	}

	// 更新的配置会载入进来
	f := newTmpFile("config.json")
	f.WriteString(`{"list": ["1", "2"]}`)
	f.Close()

	cfg, e := NewConfigStore(f.Name(), &model)

	fmt.Println("new", cfg, e)
	if e != nil {
		t.Error(0)
	}
	if !(len(model.List) == 2 && model.Hello == "world") {
		t.Error(1, model)
	}
	t.Log(model)

	// 不存在的配置会恢复为默认值
	ioutil.WriteFile(f.Name(), []byte("{}"), 0644)
	t.Log("cfg", cfg.modelPtr)
	cfg.SetAutoReload(true)
	time.Sleep(time.Second + time.Millisecond)
	if !(len(model.List) == 1 && model.Hello == "world") {
		t.Error(2)
	}

	t.Log(cfg, e, model)
}

func TestLoad(t *testing.T) {
	f := newTmpFile("config.json")
	defer f.Close()
	f.WriteString(`{"hello": "world"}`)

	m := map[string]string{}
	c, e := NewConfigStore(f.Name(), &m)
	t.Log(c, e)
	t.Log(c.Model())
}

func TestAutoReloadAndOnChange(t *testing.T) {
	f := newTmpFile("config.json")
	t.Log(f.Name())
	f.WriteString(`{"hello": "world"}`)
	f.Close()

	model := map[string]string{}
	c, e := NewConfigStore(f.Name(), &model)
	c.SetAutoReload(true)
	t.Log("loaded", c.Model(), e)

	changed := false
	c.AfterChange(func() {
		t.Log("changed", model)

		if c.Model() != &model {
			t.Error(2)
		}
		if model["hello"] != "world2" {
			t.Error(3)
		}
		changed = true
	}, false)

	ioutil.WriteFile(f.Name(), []byte(`{"hello": "world2"}`), 0644)

	time.Sleep(time.Second * 1)
	if !changed {
		t.Error(1)
	}
}

func TestStructModel(t *testing.T) {
	f := newTmpFile("config.json")
	t.Log(f.Name())
	f.WriteString(`{"hello": "world"}`)
	f.Close()

	model := struct {
		Hello string
	}{}

	c, e := NewConfigStore(f.Name(), &model)
	t.Log("loaded", c.Model(), e)

	if model.Hello != "world" {
		t.Error(1)
	}
}

func TestSave(t *testing.T) {
	f := newTmpFile("config.json")
	t.Log(f.Name())
	f.WriteString(`{"hello": "world"}`)
	f.Close()

	model := struct {
		Hello string
		List  []string
	}{
		Hello: "world",
		List:  []string{"a.com", "b.com"},
	}

	c, e := NewConfigStore(f.Name(), &model)
	c.Save()

	blob, e := ioutil.ReadFile(f.Name())
	t.Log(string(blob), e)
}

func TestYAML(t *testing.T) {
	f := newTmpFile("config.yaml")
	f.WriteString(`port: 8888
cert: ""`)

	model := struct {
		Port int
		Cert string
	}{
		9999,
		"",
	}

	c, e := NewConfigStore(f.Name(), &model)
	t.Log(c, e)
	if e != nil || model.Port != 8888 {
		t.Fail()
	}

}
