package config

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type OrderedMap struct {
	order  []string
	values map[string]any
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		order:  []string{},
		values: map[string]any{},
	}
}

func (m *OrderedMap) Len() int {
	if m == nil {
		return 0
	}
	return len(m.order)
}

func (m *OrderedMap) Keys() []string {
	if m == nil {
		return nil
	}
	keys := make([]string, len(m.order))
	copy(keys, m.order)
	return keys
}

func (m *OrderedMap) Values() map[string]any {
	if m == nil {
		return nil
	}
	values := make(map[string]any, len(m.values))
	for k, v := range m.values {
		values[k] = v
	}
	return values
}

func (m *OrderedMap) Get(key string) (any, bool) {
	if m == nil {
		return nil, false
	}
	value, ok := m.values[key]
	return value, ok
}

func (m *OrderedMap) Set(key string, value any) {
	if m.values == nil {
		m.order = []string{}
		m.values = map[string]any{}
	}
	if _, ok := m.values[key]; !ok {
		m.order = append(m.order, key)
	}
	m.values[key] = value
}

func (m *OrderedMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		*m = OrderedMap{}
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("ordered map: expected mapping node, got %v", value.Kind)
	}

	next := NewOrderedMap()
	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]
		var key string
		if err := keyNode.Decode(&key); err != nil {
			return err
		}
		var item any
		if err := valueNode.Decode(&item); err != nil {
			return err
		}
		next.Set(key, item)
	}

	*m = *next
	return nil
}

func (m *OrderedMap) MarshalYAML() (any, error) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if m == nil {
		return node, nil
	}
	for _, key := range m.order {
		valueNode := &yaml.Node{}
		if err := valueNode.Encode(m.values[key]); err != nil {
			return nil, err
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			valueNode,
		)
	}
	return node, nil
}

func (m *OrderedMap) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*m = OrderedMap{}
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return fmt.Errorf("ordered map: expected JSON object")
	}

	next := NewOrderedMap()
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("ordered map: expected string key")
		}
		var value any
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		next.Set(key, value)
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}

	*m = *next
	return nil
}

func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}

	var buffer bytes.Buffer
	buffer.WriteByte('{')
	for i, key := range m.order {
		if i > 0 {
			buffer.WriteByte(',')
		}
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		valueBytes, err := json.Marshal(m.values[key])
		if err != nil {
			return nil, err
		}
		buffer.Write(keyBytes)
		buffer.WriteByte(':')
		buffer.Write(valueBytes)
	}
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}
