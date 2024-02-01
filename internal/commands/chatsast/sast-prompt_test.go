package chatsast

import (
	"testing"
)

func TestAddNewlinesIfNecessaryNoNewlines(t *testing.T) {
	input := confidence + " 35 " + explanation + " this is a short explanation." + fix + " a fixed snippet"
	expected := confidence + " 35 \n" + explanation + " this is a short explanation.\n" + fix + " a fixed snippet"

	output := getActual(input, t)

	if output[len(output)-1] != expected {
		t.Errorf("Expected %q, but got %q", expected, output)
	}
}

func TestAddNewlinesIfNecessarySomeNewlines(t *testing.T) {
	input := confidence + " 35 " + explanation + " this is a short explanation.\n  " + fix + " a fixed snippet"
	expected := confidence + " 35 \n" + explanation + " this is a short explanation.\n  " + fix + " a fixed snippet"

	output := getActual(input, t)

	if output[len(output)-1] != expected {
		t.Errorf("Expected %q, but got %q", expected, output)
	}
}

func TestAddNewlinesIfNecessaryAllNewlines(t *testing.T) {
	input := confidence + " 35\n " + explanation + " this is a short explanation.\n  " + fix + " a fixed snippet"
	expected := input

	output := getActual(input, t)

	if output[len(output)-1] != expected {
		t.Errorf("Expected %q, but got %q", expected, output)
	}
}

func getActual(input string, t *testing.T) []string {
	someText := "some text"
	response := []string{someText, someText, input}
	output := AddNewlinesIfNecessary(response)
	for i := 0; i < len(output)-1; i++ {
		if output[i] != response[i] {
			t.Errorf("All strings except last expected to stay the same")
		}
	}
	return output
}
