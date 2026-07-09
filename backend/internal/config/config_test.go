package config

import (
	"os"
	"testing"
)

func TestFirstNonEmpty(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"", "", "a"}, "a"},
		{[]string{"x", "y"}, "x"},
		{[]string{"", ""}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := firstNonEmpty(c.in...); got != c.want {
			t.Errorf("firstNonEmpty(%v) = %q, beklenen %q", c.in, got, c.want)
		}
	}
}

func TestGetenv(t *testing.T) {
	const key = "M2M_TEST_GETENV"
	t.Setenv(key, "değer")
	if got := getenv(key, "yedek"); got != "değer" {
		t.Errorf("dolu env için değer bekleniyordu, gelen %q", got)
	}
	os.Unsetenv(key)
	if got := getenv(key, "yedek"); got != "yedek" {
		t.Errorf("boş env için yedek bekleniyordu, gelen %q", got)
	}
}
