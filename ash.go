package ashway

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

type AshValidator func(value string) (name string, ok bool)

type AshFetcher func(id string) (value json.RawMessage, err error)

type AshAlarm func(err error)

func NewAsh(valid AshValidator) *Ash {
	return &Ash{valid: valid, fetchers: make(map[string]AshFetcher)}
}

type Ash struct {
	valid    AshValidator
	alarm    AshAlarm
	fetchers map[string]AshFetcher
}

func (ash *Ash) RegisterGetter(name string, fetcher AshFetcher) {
	ash.fetchers[name] = fetcher
}

func (ash *Ash) SetAlarm(alarm AshAlarm) {
	ash.alarm = alarm
}

func (ash *Ash) Burn(in json.RawMessage) (out json.RawMessage, err error) {
	if in == nil || len(in) == 0 {
		err = io.EOF
		return
	}
	if in[0] == '{' {
		m0 := make(map[string]interface{})
		err = json.Unmarshal(in, &m0)
		if err != nil {
			return
		}
		m1, _, walkErr := ash.walkMap(m0)
		if walkErr != nil {
			err = walkErr
			return
		}
		out, err = json.Marshal(m1)
		return
	}

	if in[0] == '[' {
		s0 := make([]interface{}, 0, 1)
		err = json.Unmarshal(in, &s0)
		if err != nil {
			return
		}
		s1, _, _, walkErr := ash.walkSlice("", s0)
		if walkErr != nil {
			err = walkErr
			return
		}
		out, err = json.Marshal(s1)
		return
	}

	return
}

func (ash *Ash) walkMap(m0 map[string]interface{}) (map[string]interface{}, bool, error) {
	fetched := false
	for k, v := range m0 {
		typeOfValue := reflect.TypeOf(v)
		kindOfValue := typeOfValue.Kind()
		if kindOfValue == reflect.Map {
			m1, ok := reflect.ValueOf(v).Interface().(map[string]interface{})
			if ok {
				_v, _, err := ash.walkMap(m1)
				if err != nil {
					return nil, false, err
				}
				m0[k] = _v
			}
			continue
		}
		if kindOfValue == reflect.Slice {
			s1, ok := reflect.ValueOf(v).Interface().([]interface{})
			if ok {
				_v, name, fetched, err := ash.walkSlice(k, s1)
				if err != nil {
					return nil, false, err
				}
				if fetched {
					m0[name] = _v
				} else {
					m0[k] = _v
				}
			}
			continue
		}

		name := ""
		mapped := false
		if len(k) > 0 {
			name, mapped = ash.valid(k)
		}
		if mapped {
			if fetcher, has := ash.fetchers[name]; has {
				id := ""
				if kindOfValue == reflect.String {
					id = reflect.ValueOf(v).String()
				} else if reflect.TypeOf(v).Kind() == reflect.Float64 {
					id = strconv.Itoa(int(reflect.ValueOf(v).Float()))
				} else {
					return nil, false, ErrTargetIdIsNotStringOrInt
				}
				target, err := fetcher(id)
				if err != nil {
					return nil, false, err
				}

				if target == nil || len(target) == 0 {
					if ash.alarm != nil {
						ash.alarm(fmt.Errorf("fetch nothing by %s's id", name))
					}
					continue
				}
				if target[0] != '{' {
					if ash.alarm != nil {
						ash.alarm(fmt.Errorf("fetch not a json object by %s's id", name))
					}
					continue
				}
				nv := make(map[string]interface{})
				if unmarshalErr := json.Unmarshal(target, &nv); unmarshalErr != nil {
					if ash.alarm != nil {
						ash.alarm(fmt.Errorf("fetch a bad json object by %s's id", name))
					}
					continue
				}
				m0[name] = nv
				fetched = true
			}
		}
	}
	return m0, fetched, nil
}

func (ash *Ash) walkSlice(key string, s0 []interface{}) ([]interface{}, string, bool, error) {
	name := ""
	mapped := false
	var s1 []interface{} = nil
	if len(key) > 0 {
		name, mapped = ash.valid(key)
		s1 = make([]interface{}, 0, 1)
	}
	for i, v := range s0 {
		rkv := reflect.TypeOf(v).Kind()
		if rkv == reflect.Map {
			m1, ok := reflect.ValueOf(v).Interface().(map[string]interface{})
			if ok {
				_v, _, err := ash.walkMap(m1)
				if err != nil {
					return nil, "", false, err
				}
				s0[i] = _v
			}
			continue
		}
		if rkv == reflect.Slice {
			s1, ok := reflect.ValueOf(v).Interface().([]interface{})
			if ok {
				_s0, _, _, err := ash.walkSlice(key, s1)
				if err != nil {
					return nil, "", false, err
				}
				s0 = _s0
			}
			continue
		}
		if key == "" {
			continue
		}
		if mapped {
			if fetcher, has := ash.fetchers[name]; has {
				id := ""
				if rkv == reflect.String {
					id = reflect.ValueOf(v).String()
				} else if reflect.TypeOf(v).Kind() == reflect.Float64 {
					id = strconv.Itoa(int(reflect.ValueOf(v).Float()))
				} else {
					return nil, "", false, ErrTargetIdIsNotStringOrInt
				}
				target, err := fetcher(id)
				if err != nil {
					return nil, "", false, err
				}

				if target == nil || len(target) == 0 {
					if ash.alarm != nil {
						ash.alarm(fmt.Errorf("fetch nothing by %s's id", name))
					}
					continue
				}
				if target[0] != '{' {
					if ash.alarm != nil {
						ash.alarm(fmt.Errorf("fetch not a json object by %s's id", name))
					}
					continue
				}
				nv := make(map[string]interface{})
				if unmarshalErr := json.Unmarshal(target, &nv); unmarshalErr != nil {
					if ash.alarm != nil {
						ash.alarm(fmt.Errorf("fetch a bad json object by %s's id", name))
					}
					continue
				}
				s1 = append(s1, nv)
			}
		}
	}
	if mapped {
		return s1, name + "s", true, nil
	}
	return s0, "", false, nil
}

var (
	ErrTargetIdIsNotStringOrInt   = errors.New("target id is not string or int")
	ErrBurnFailedCauseEmptySource = errors.New("burn failed, cause source json bytes is empty")
)
