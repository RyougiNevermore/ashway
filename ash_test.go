package ashway_test

import (
	"encoding/json"
	"fmt"
	"github.com/pharosnet/ashway"
	"strings"
	"testing"
	"time"
)

type Course struct {
	Ident      string     `json:"ident"`
	Ids        [][]string `json:"ids"`
	Id         int64      `json:"id"`
	Name       string     `json:"name"`
	TeacherId  int        `json:"teacher_id"`
	StudentIds []string   `json:"student_ids"`
	Lessons    []Lesson   `json:"lessons"`
	Lesson     Lesson     `json:"lesson"`
}

type User struct {
	Id   string    `json:"id"`
	Name string    `json:"name"`
	Age  int       `json:"age"`
	Day  time.Time `json:"day"`
}

type Lesson struct {
	StudentId string `json:"student_id"`
	Name      string `json:"name"`
}

func userValidator(value string) (name string, ok bool) {
	idx := 0
	isLast := false
	if idx = strings.Index(value, "_ids"); idx > 0 {
		isLast = len(value[idx:]) == 4
		name = strings.ToLower(value[0:idx])
	} else if idx = strings.Index(value, "_id"); idx > 0 {
		isLast = len(value[idx:]) == 3
		name = strings.ToLower(value[0:idx])
	}
	if !isLast || len(name) == 0 {
		return
	}
	ok = true
	return
}

func userFetcher(id string) (value json.RawMessage, err error) {
	if id == "2" {
		return
	}
	u := User{
		Id:   id,
		Name: fmt.Sprintf("name_%s", id),
		Age:  11,
		Day:  time.Now(),
	}
	value, err = json.Marshal(&u)
	return
}

func TestNewAsh(t *testing.T) {
	ash := ashway.NewAsh(userValidator)
	ash.RegisterGetter("teacher", userFetcher)
	ash.RegisterGetter("student", userFetcher)
	course := Course{
		Ident:      "sss",
		Ids:        [][]string{{"sss"}},
		Id:         111,
		Name:       "c1",
		TeacherId:  123456,
		StudentIds: []string{"1", "2", "3"},
		Lessons:    []Lesson{{StudentId: "11", Name: "l11"}},
		Lesson:     Lesson{StudentId: "11", Name: "l11"},
	}

	srcp, _ := json.Marshal(&course)

	out, err := ash.Burn(srcp)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(out))
}
