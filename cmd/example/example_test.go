package main

import (
	"errors"
	"os"
	"testing"

	"github.com/YuriyNasretdinov/golang-soft-mocks"
)

func TestSoft(t *testing.T) {
	resetMethod(t, false)
}

func TestSoftAll(t *testing.T) {
	resetMethod(t, true)
}

func resetMethod(t *testing.T, all bool) {
	soft.Mock(os.Open, func(filename string) (*os.File, error) {
		return nil, errors.New("Cannot open files!")
	})

	if _, err := os.Open(os.DevNull); err == nil {
		t.Fatalf("Must be error opening dev null!")
	}

	if all {
		soft.ResetAll()
	} else {
		soft.Reset(os.Open)
	}

	if _, err := os.Open(os.DevNull); err != nil {
		t.Fatalf("Must be no errors opening dev null after mock reset!")
	}
}
