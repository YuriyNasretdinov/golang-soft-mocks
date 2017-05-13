package main

import (
	"errors"
	"os"
	"testing"

	"github.com/YuriyNasretdinov/golang-soft-mocks"
)

func TestSoft(t *testing.T) {
	soft.Mock(os.Open, func(filename string) (*os.File, error) {
		return nil, errors.New("Cannot open files!")
	})

	if _, err := os.Open(os.DevNull); err != nil {
		t.Fatalf("Must be no errors opening dev null!")
	}
}
