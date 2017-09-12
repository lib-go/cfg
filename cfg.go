package cfg

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type ConfigStore struct {
	path             string
	fileInfo         os.FileInfo
	enableAutoReload bool
	modelPtr         interface{} // pointer to model
	defaultData      []byte
	afterChange      func()
	format           string
}

func NewConfigStore(path string, modelPtr interface{}) (c *ConfigStore, e error) {
	rv := reflect.ValueOf(modelPtr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		e = fmt.Errorf("modelPtr must be pointer")
		return
	}
	c = &ConfigStore{
		path:     path,
		modelPtr: modelPtr,
	}
	c.defaultData, e = dumps(c.Format(), c.modelPtr)

	if e = c.Load(); e != nil {
		return nil, e
	}

	return
}

func MustBindConfig(path string, modelPtr interface{}, onChange func()) (c *ConfigStore) {
	var e error
	if c, e = NewConfigStore(path, modelPtr); e != nil {
		panic(e)
	}

	if onChange != nil {
		c.SetAutoReload(true)
		c.AfterChange(onChange, true)
	}
	return c
}

func (c *ConfigStore) Format() string {
	if c.format == "" {
		c.format = strings.Replace(strings.ToLower(filepath.Ext(c.path)), ".", "", 1)
	}
	return c.format
}

func (c *ConfigStore) String() string {
	return "[Config]"
}

func (c *ConfigStore) Load() (e error) {
	var file *os.File

	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		file, e = os.Create(c.path)
	} else {
		file, e = os.Open(c.path)
	}
	defer file.Close()

	if e != nil {
		return
	}

	blob, e := ioutil.ReadAll(file)
	if e != nil {
		return
	}

	info, _ := os.Stat(c.path)
	c.fileInfo = info

	e = loads(c.Format(), c.defaultData, c.modelPtr) // 配置变更以后,missing key应该恢复为默认值
	if len(blob) != 0 {
		e = loads(c.Format(), blob, c.modelPtr)
	}

	return
}

func (c *ConfigStore) SetAutoReload(enabled bool) {
	if c.enableAutoReload == false && enabled == true {
		go c.startAutoReloading()
	}
	c.enableAutoReload = enabled
}

func (c *ConfigStore) AfterChange(afterChange func(), applyNow bool) {
	c.afterChange = afterChange
	if afterChange != nil {
		c.SetAutoReload(true)

		if applyNow {
			afterChange()
		}
	}
}

func (c *ConfigStore) startAutoReloading() {
	for {
		if !c.enableAutoReload {
			return
		}
		info, err := os.Stat(c.path)

		if err == nil {
			if info.ModTime() != c.fileInfo.ModTime() || info.Size() != c.fileInfo.Size() {
				c.Load()
				c.EmitChange()
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func (c *ConfigStore) EmitChange() {
	if c.afterChange != nil {
		c.afterChange()
	}
}

func (c *ConfigStore) Model() interface{} {
	return c.modelPtr
}

func (c *ConfigStore) Save() (e error) {
	var output []byte
	output, e = dumps(c.Format(), c.modelPtr)
	if e != nil {
		return e
	}
	e = ioutil.WriteFile(c.path, output, 0644)
	return e
}

func loads(format string, chunk []byte, v interface{}) (e error) {
	switch format {
	case "json":
		e = json.Unmarshal(chunk, v)
	case "yaml":
		e = yaml.Unmarshal(chunk, v)
	default:
		e = fmt.Errorf("format not supported: %s", format)
	}
	return
}

func dumps(format string, in interface{}) (output []byte, e error) {
	switch format {
	case "json":
		output, e = json.MarshalIndent(in, " ", "  ")
	case "yaml":
		output, e = yaml.Marshal(in)
	default:
		e = fmt.Errorf("format not supported: %s", format)
	}
	return
}
