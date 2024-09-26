package go_cfg

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type ConfigFormat string

const (
	FormatJSON ConfigFormat = "json"
	FormatYAML ConfigFormat = "yaml"
)

type OnUpdateFunc func(changed bool)

type ConfigStore struct {
	path        string
	modelPtr    interface{} // pointer to model
	format      ConfigFormat
	defaultData []byte // 初始化时modelPtr的初始值，作为文件中missing的值的fallback

	isWatching bool
	onUpdate   OnUpdateFunc

	fileInfo os.FileInfo
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

	ext := strings.Replace(strings.ToLower(filepath.Ext(c.path)), ".", "", 1)
	switch ext {
	case "json":
		c.format = FormatJSON
	case "yaml":
		c.format = FormatYAML
	default:
		return nil, fmt.Errorf("unknown format: %s", ext)
	}

	c.defaultData, e = dumps(c.format, c.modelPtr)

	if e = c.Load(); e != nil {
		return nil, e
	}

	return
}

func MustBindConfig(path string, modelPtr interface{}, onUpdate OnUpdateFunc) (c *ConfigStore) {
	var e error
	if c, e = NewConfigStore(path, modelPtr); e != nil {
		panic(e)
	}
	c.Watch(onUpdate, true)
	return c
}

func (c *ConfigStore) Format() ConfigFormat {
	return c.format
}

func (c *ConfigStore) Model() interface{} {
	return c.modelPtr
}

func (c *ConfigStore) String() string {
	return "[Config]"
}

func (c *ConfigStore) Load() (e error) {
	var fileInfo os.FileInfo
	var file *os.File
	var blob []byte

	if fileInfo, e = os.Stat(c.path); os.IsNotExist(e) {
		return fs.ErrNotExist
	}
	if file, e = os.Open(c.path); e != nil {
		return
	}
	if blob, e = io.ReadAll(file); e != nil {
		return
	}

	c.fileInfo = fileInfo

	// 配置变更以后,missing key应该恢复为初始值
	e = loads(c.Format(), c.defaultData, c.modelPtr)
	if len(blob) != 0 {
		e = loads(c.Format(), blob, c.modelPtr)
	}

	return
}

func (c *ConfigStore) Save() (e error) {
	var output []byte
	if output, e = dumps(c.Format(), c.modelPtr); e != nil {
		return
	}
	return os.WriteFile(c.path, output, 0644)
}

func (c *ConfigStore) Watch(fn OnUpdateFunc, callUpdateNow bool) {
	c.onUpdate = fn
	if callUpdateNow && fn != nil {
		fn(false)
	}

	if !c.isWatching {
		c.isWatching = true
		go c.startWatching()
	}
}

func (c *ConfigStore) isConfigFileChanged() (changed bool) {
	info, err := os.Stat(c.path)
	if err == nil {
		if info.ModTime() != c.fileInfo.ModTime() || info.Size() != c.fileInfo.Size() {
			return true
		}
	}
	return false
}

func (c *ConfigStore) startWatching() {
	for {
		if !c.isWatching {
			return
		}

		if c.isConfigFileChanged() {
			if e := c.Load(); e != nil {
				log.Println("config file changed, but load failed:", e)
			} else if c.onUpdate != nil {
				c.onUpdate(true)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func (c *ConfigStore) StopWatch() {
	c.isWatching = false
}

func loads(format ConfigFormat, chunk []byte, v interface{}) (e error) {
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

func dumps(format ConfigFormat, in interface{}) (output []byte, e error) {
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
