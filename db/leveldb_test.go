package db

import (
	"fmt"
	"testing"
)

func TestNewLDBDataBase(t *testing.T) {
	db, err := NewLDBDataBase("test")
	if err != nil {
		log.Error(err)
	}
	//_ = db.Put([]byte("1234"),[]byte("lowes"))
	data, _ := db.Get([]byte("1234"))
	fmt.Println(string(data))
}
