package onectechcommon

import "reflect"

type metrics struct {
	allMaps []map[string]*storage

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

func newMetrics() *metrics {
	m := &metrics{}
	v := reflect.ValueOf(m).Elem()
	t := v.Type()
	mapType := reflect.TypeOf(map[string]*storage{})
	var maps []map[string]*storage

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		if fieldName == "allMaps" {
			continue
		}

		if field.Kind() == reflect.Map && field.Type().AssignableTo(mapType) {
			newMap := reflect.MakeMap(field.Type())
			field.Set(newMap)
			mp := field.Interface().(map[string]*storage)
			m.add(mp, "Total", 0)
			maps = append(maps, mp)
		}
	}
	m.allMaps = maps
	return m
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
