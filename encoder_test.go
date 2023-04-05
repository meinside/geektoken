package geektoken

import (
	"bufio"
	"os"
	"testing"
)

const sampleFilepath = "data/samples.txt"

// read sample texts
func readSamples() (samples []string, err error) {
	samples = []string{}

	var file *os.File
	if file, err = os.Open(sampleFilepath); err == nil {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for {
			if scanner.Scan() {
				samples = append(samples, scanner.Text())
			} else {
				break
			}
		}
	}

	return samples, err
}

// check if given []int values equal
func equals(a, b []int) bool {
	if len(a) == len(b) {
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	return false
}

func TestEncodingAndDecoding(t *testing.T) {
	if samples, err := readSamples(); err == nil {
		for _, name := range []EncodingName{EncodingCl100kBase, EncodingR50kBase, EncodingP50kBase, EncodingP50kEdit} {
			if tokenizer, err := GetTokenizerWithEncoding(name); err == nil {
				for _, sample := range samples {
					if encoded, err := tokenizer.Encode(sample, nil, nil); err != nil {
						t.Errorf("failed encoding sample '%s': %s", sample, err)
					} else {
						decoded := tokenizer.Decode(encoded)
						if decoded != sample {
							t.Errorf("failed decoding with model: %v, sample: '%s', decoded: '%s'", name, sample, decoded)
						}
					}
				}
			} else {
				t.Errorf("failed to get tokenizer with model name: %v, error: %s", name, err)
			}
		}
	} else {
		t.Errorf("failed to read samples: %s", err)
	}
}

func TestEncodingWithCustomAllowedSet(t *testing.T) {
	name := EncodingCl100kBase
	inputText := "Some Text<|endofprompt|>"
	allowedSpecialTokens := []string{"<|endofprompt|>"}
	expected := []int{8538, 2991, 100276}

	if tokenizer, err := GetTokenizerWithEncoding(name); err == nil {
		if encoded, err := tokenizer.Encode(inputText, allowedSpecialTokens, nil); err == nil {
			if !equals(encoded, expected) {
				t.Errorf("failed with model: %v, input text: '%s', encoded: %+v, expected: %+v", name, inputText, encoded, expected)
			}
		} else {
			t.Errorf("failed to encode input text '%s': %s", inputText, err)
		}

	} else {
		t.Errorf("no such model: %s", string(name))
	}
}

func TestEncodingFailsWithInvalidInputDefaultSpecial(t *testing.T) {
	name := EncodingCl100kBase
	inputText := "Some Text<|endofprompt|>"

	if tokenizer, err := GetTokenizerWithEncoding(name); err == nil {
		if _, err := tokenizer.Encode(inputText, nil, nil); err == nil {
			t.Errorf("should fail with invalid input, but did not.")
		}
	} else {
		t.Errorf("no such model: %s", string(name))
	}
}

func TestEncodingFailsWithInvalidInputCustomDisallowed(t *testing.T) {
	name := EncodingCl100kBase
	inputText := "Some Text"
	disallowedSpecialTokens := []string{"Some"}

	if tokenizer, err := GetTokenizerWithEncoding(name); err == nil {
		if _, err := tokenizer.Encode(inputText, nil, disallowedSpecialTokens); err == nil {
			t.Errorf("should fail with invalid input, but did not.")
		}
	} else {
		t.Errorf("no such model: %s", string(name))
	}
}
