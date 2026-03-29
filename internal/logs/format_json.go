package logs

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const defaultJSONTimeLayout = "2006-01-02T15:04:05.000Z07:00"

type JSONFormatter struct {
	cfg formatterConfig
}

func NewJSONFormatter(opts ...FormatterOption) *JSONFormatter {
	cfg := applyFormatterOpts(formatterConfig{timeLayout: defaultJSONTimeLayout}, opts)
	return &JSONFormatter{cfg: cfg}
}

func (f *JSONFormatter) Format(entry Entry) ([]byte, error) {
	obj := newOrderedMap()

	obj.set("time", entry.Time.Format(f.cfg.timeLayout))
	obj.set("level", entry.Level.String())
	obj.set("msg", entry.Message)

	allKVs := mergeKVPairs(entry.Bound, entry.Args)
	f.writeKVPairs(obj, allKVs)

	if entry.Error != nil {
		info := ExtractError(entry.Error)
		if info.Code != "" {
			obj.set("err_code", info.Code)
		} else {
			obj.setRaw("err_code", []byte("null"))
		}
		obj.set("err_msg", info.Message)
		if info.Stack != "" {
			obj.set("err_stack", info.Stack)
		} else {
			obj.setRaw("err_stack", []byte("null"))
		}
	}

	data := obj.marshal()
	data = append(data, '\n')
	return data, nil
}

func (f *JSONFormatter) writeKVPairs(obj *orderedMap, kvs []any) {
	if len(kvs) == 0 {
		return
	}
	if len(kvs)%2 != 0 {
		internalWarn("odd number of args, using index keys", "count", len(kvs))
		for i, v := range kvs {
			key := "_arg" + strconv.Itoa(i)
			obj.set(key, f.encodeValue(v))
		}
		return
	}
	for i := 0; i < len(kvs)-1; i += 2 {
		key := fmt.Sprintf("%v", kvs[i])
		val := kvs[i+1]
		obj.setDedup(key, f.encodeValue(val))
	}
}

func (f *JSONFormatter) encodeValue(v any) any {
	switch val := v.(type) {
	case string:
		return val
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return val
	case json.RawMessage:
		if json.Valid(val) {
			return rawJSON(val)
		}
		internalWarn("invalid json.RawMessage, degrading to string", "value", string(val))
		return string(val)
	case json.Marshaler:
		data, err := val.MarshalJSON()
		if err != nil {
			internalWarn("json.Marshaler failed, degrading to string", "error", err)
			return fmt.Sprintf("%+v", v)
		}
		if !json.Valid(data) {
			internalWarn("json.Marshaler produced invalid JSON, degrading to string", "value", string(data))
			return fmt.Sprintf("%+v", v)
		}
		return rawJSON(data)
	default:
		return fmt.Sprintf("%+v", v)
	}
}

// rawJSON 是已序列化的 JSON bytes，marshal 時直接嵌入，不再重新 encode。
type rawJSON []byte

func (r rawJSON) MarshalJSON() ([]byte, error) { return r, nil }

type orderedMap struct {
	keys   []string
	values map[string]any
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]any)}
}

func (m *orderedMap) set(key string, val any) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = val
}

func (m *orderedMap) setRaw(key string, raw []byte) {
	m.set(key, rawJSON(raw))
}

func (m *orderedMap) setDedup(key string, val any) {
	if _, exists := m.values[key]; !exists {
		m.set(key, val)
		return
	}
	internalWarn("duplicate key in log args", "key", key)
	for i := 2; ; i++ {
		candidate := key + "_" + strconv.Itoa(i)
		if _, exists := m.values[candidate]; !exists {
			m.set(candidate, val)
			return
		}
	}
}

func (m *orderedMap) marshal() []byte {
	var buf []byte
	buf = append(buf, '{')
	for i, key := range m.keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyBytes, _ := json.Marshal(key)
		buf = append(buf, keyBytes...)
		buf = append(buf, ':')
		valBytes, err := json.Marshal(m.values[key])
		if err != nil {
			valBytes, _ = json.Marshal(fmt.Sprintf("%+v", m.values[key]))
		}
		buf = append(buf, valBytes...)
	}
	buf = append(buf, '}')
	return buf
}
