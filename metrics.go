package onectechcommon

import (
	"encoding/json"
	"reflect"
	"strings"
)

type metrics struct {
	allMaps  []map[string]*storage
	nameMaps []string
	indexMap map[string]int

	DbLock         map[string]*storage `json:"dbLock"`
	CallCount      map[string]*storage `json:"callCount"`
	CallDuration   map[string]*storage `json:"callDuration"`
	CallCpu        map[string]*storage `json:"callCpu"`
	CallMemoryPeak map[string]*storage `json:"callMemoryPeak"`
	CallInBytes    map[string]*storage `json:"callInBytes"`
	CallOutBytes   map[string]*storage `json:"callOutBytes"`
	TTimeout       map[string]*storage `json:"tTimeout"`
	TDeadlock      map[string]*storage `json:"tDeadlock"`
	Excp           map[string]*storage `json:"excp"`
	BadAlloc       map[string]*storage `json:"badAlloc"`
}

type metricItem struct {
	KEY   string `json:"KEY"`
	PARAM string `json:"PARAM"`
	VALUE int64  `json:"VALUE"`
}

func newMetrics() *metrics {
	m := &metrics{}
	v := reflect.ValueOf(m).Elem()
	t := v.Type()
	mapType := reflect.TypeOf(map[string]*storage{})
	var maps []map[string]*storage
	var names []string
	indexes := make(map[string]int)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name
		fieldType := t.Field(i)

		switch fieldName {
		case "allMaps", "nameMaps", "indexMap":
			continue
		}

		if field.Kind() == reflect.Map && field.Type().AssignableTo(mapType) {
			newMap := reflect.MakeMap(field.Type())
			field.Set(newMap)
			mp := field.Interface().(map[string]*storage)
			m.add(mp, "Total", 0)
			maps = append(maps, mp)

			tag := fieldType.Tag.Get("json")
			if tag == "" || tag == "-" {
				tag = fieldName
			} else {
				if idx := strings.Index(tag, ","); idx != -1 {
					tag = tag[:idx]
				}
			}
			names = append(names, tag)

			indexes[tag] = len(maps) - 1
		}
	}
	m.allMaps = maps
	m.nameMaps = names
	m.indexMap = indexes
	return m
}

func (m *metrics) getValue(key, param string) int64 {
	index := m.indexMap[key]
	return m.allMaps[index][param].value
}

func (m *metrics) setValue(key, param string, value int64) {
	index := m.indexMap[key]
	m.allMaps[index][param].value = value
}

func (m *metrics) add(store map[string]*storage, key string, value int64) {
	s := store[key]
	if s == nil {
		s = &storage{}
		store[key] = s
	}
	s.add(value)
}

func (m *metrics) clear() {
	for _, mMap := range m.allMaps {
		for k := range mMap {
			mMap[k].clear()
		}
	}
}

func (m *metrics) toJSON(zeroValues bool) ([]byte, error) {
	var items []metricItem

	for i, mp := range m.allMaps {
		if i >= len(m.nameMaps) {
			break
		}
		metricName := m.nameMaps[i]
		for process, st := range mp {
			val := st.value
			if zeroValues {
				val = 0
			}
			items = append(items, metricItem{
				KEY:   metricName,
				PARAM: process,
				VALUE: val,
			})
		}
	}
	return json.Marshal(items)
}
