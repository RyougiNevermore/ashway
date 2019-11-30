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

type AshFetcher func(name string, id string) (value json.RawMessage, err error)

type AshAlarm func(err error)

func NewAsh(replaced bool, valid AshValidator, fetcher AshFetcher) *Ash {
	return &Ash{replaced: replaced, valid: valid, fetcher: fetcher}
}

type Ash struct {
	replaced bool
	valid    AshValidator
	alarm    AshAlarm
	fetcher  AshFetcher
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
		m1, walkErr := ash.walkMap(m0)
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
		s1, walkErr := ash.walkSlice(s0)
		if walkErr != nil {
			err = walkErr
			return
		}
		out, err = json.Marshal(s1)
		return
	}

	err = ErrNotBurnJsonBytes

	return
}

func (ash *Ash) walkMap(m0 map[string]interface{}) (map[string]interface{}, error) {
	for k, v := range m0 {
		typeOfValue := reflect.TypeOf(v)
		kindOfValue := typeOfValue.Kind()
		if kindOfValue == reflect.Map {
			m1, ok := reflect.ValueOf(v).Interface().(map[string]interface{})
			if ok {
				_v, err := ash.walkMap(m1)
				if err != nil {
					return nil, err
				}
				m0[k] = _v
			}
			continue
		}
		if kindOfValue == reflect.Slice {

			s1, ok := reflect.ValueOf(v).Interface().([]interface{})
			if ok {
				_v, name, err := ash.walkSliceWithKey(k, s1)
				if err != nil {
					return nil, err
				}
				m0[name] = _v
				if name != k && ash.replaced {
					delete(m0, k)
				}
			}
			continue
		}

		name, mapped := ash.valid(k)
		if mapped {
			id := ""
			if kindOfValue == reflect.String {
				id = reflect.ValueOf(v).String()
			} else if reflect.TypeOf(v).Kind() == reflect.Float64 {
				id = strconv.Itoa(int(reflect.ValueOf(v).Float()))
			} else {
				return nil, ErrTargetIdIsNotStringOrInt
			}
			targetValue, err := ash.fetcher(name, id)
			if err != nil {
				return nil, err
			}

			if targetValue == nil || len(targetValue) == 0 {
				if ash.alarm != nil {
					ash.alarm(fmt.Errorf("fetch nothing by %s's id", name))
				}
				continue
			}
			if targetValue[0] != '{' {
				if ash.alarm != nil {
					ash.alarm(fmt.Errorf("fetch not a json object by %s's id", name))
				}
				continue
			}
			nv := make(map[string]interface{})
			if unmarshalErr := json.Unmarshal(targetValue, &nv); unmarshalErr != nil {
				if ash.alarm != nil {
					ash.alarm(fmt.Errorf("fetch a bad json object by %s's id", name))
				}
				continue
			}
			m0[name] = nv
			if ash.replaced {
				delete(m0, k)
			}
		}
	}
	return m0, nil
}

func (ash *Ash) walkSliceWithKey(key string, s0 []interface{}) ([]interface{}, string, error) {
	if s0 == nil || len(s0) == 0 {
		return s0, key, nil
	}
	if len(key) == 0 {
		return s0, key, nil
	}

	name, mapped := ash.valid(key)

	ns0 := make([]interface{}, len(s0))
	copy(ns0, s0)

	for i, v := range ns0 {
		rkv := reflect.TypeOf(v).Kind()
		if rkv == reflect.Map {
			m1, ok := reflect.ValueOf(v).Interface().(map[string]interface{})
			if ok {
				_v, err := ash.walkMap(m1)
				if err != nil {
					return ns0, key, err
				}
				ns0[i] = _v
			}
			continue
		}
		if rkv == reflect.Slice {
			s1, ok := reflect.ValueOf(v).Interface().([]interface{})
			if ok {
				_s0, err := ash.walkSlice(s1)
				if err != nil {
					return ns0, key, err
				}
				ns0[i] = _s0
			}
			continue
		}

		if mapped {
			id := ""
			if rkv == reflect.String {
				id = reflect.ValueOf(v).String()
			} else if reflect.TypeOf(v).Kind() == reflect.Float64 {
				id = strconv.Itoa(int(reflect.ValueOf(v).Float()))
			} else {
				return ns0, key, ErrTargetIdIsNotStringOrInt
			}
			targetValue, err := ash.fetcher(name, id)
			if err != nil {
				continue
			}
			if targetValue == nil || len(targetValue) == 0 {
				if ash.alarm != nil {
					ash.alarm(fmt.Errorf("fetch nothing by %s's id", name))
				}
				continue
			}

			if targetValue[0] != '{' {
				if ash.alarm != nil {
					ash.alarm(fmt.Errorf("fetch not a json object by %s's id", name))
				}
				continue
			}

			nv := make(map[string]interface{})
			if unmarshalErr := json.Unmarshal(targetValue, &nv); unmarshalErr != nil {
				if ash.alarm != nil {
					ash.alarm(fmt.Errorf("fetch a bad json object by %s's id", name))
				}
				continue
			}
			ns0[i] = nv
		}
	}

	if mapped {
		key = name
	}

	return ns0, key, nil
}

func (ash *Ash) walkSlice(s0 []interface{}) ([]interface{}, error) {
	for i, v := range s0 {
		rkv := reflect.TypeOf(v).Kind()
		if rkv == reflect.Map {
			m1, ok := reflect.ValueOf(v).Interface().(map[string]interface{})
			if ok {
				_v, err := ash.walkMap(m1)
				if err != nil {
					return s0, err
				}
				s0[i] = _v
			}
			continue
		}
		if rkv == reflect.Slice {
			s1, ok := reflect.ValueOf(v).Interface().([]interface{})
			if ok {
				_s0, err := ash.walkSlice(s1)
				if err != nil {
					return s0, err
				}
				s0[i] = _s0
			}
			continue
		}

	}
	return s0, nil
}

var (
	ErrTargetIdIsNotStringOrInt = errors.New("target id is not string or int")
	ErrNotBurnJsonBytes         = errors.New("the source is not json bytes")
)
