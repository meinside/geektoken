package geektoken

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"
)

//go:embed data/cl100k_base.tiktoken
var cl100kBaseLines string

//go:embed data/p50k_base.tiktoken
var p50kBaseLines string

//go:embed data/r50k_base.tiktoken
var r50kBaseLines string

type EncodingName string
type ModelName string

const (
	EncodingCl100kBase EncodingName = "cl100k_base"
	EncodingP50kBase   EncodingName = "p50k_base"
	EncodingP50kEdit   EncodingName = "p50k_edit"
	EncodingR50kBase   EncodingName = "r50k_base"

	ModelGPT4                ModelName = "gpt-4"
	ModelGPT35Turbo          ModelName = "gpt-3.5-turbo"
	ModelTextEmbeddingAda002 ModelName = "text-embedding-ada-002"
	ModelTextDavinci002      ModelName = "text-davinci-002"
	ModelTextDavinci003      ModelName = "text-davinci-003"
	ModelGPT2                ModelName = "gpt2"
	ModelDavinci             ModelName = "davinci"
)

type Params struct {
	ExplicitNVocab *int
	PatStr         string
	MergeableRanks map[string]int
	SpecialTokens  map[string]int
}

const (
	EndOfText   = "<|endoftext|>"
	FimPrefix   = "<|fim_prefix|>"
	FimMiddle   = "<|fim_middle|>"
	FimSuffix   = "<|fim_suffix|>"
	EndOfPrompt = "<|endofprompt|>"
)

func getEncodingParams(name EncodingName) (Params, error) {
	switch name {
	case EncodingR50kBase:
		return r50kBase(), nil
	case EncodingP50kBase:
		return p50kBase(), nil
	case EncodingP50kEdit:
		return p50kEdit(), nil
	case EncodingCl100kBase:
		return cl100kBase(), nil
	}

	return Params{}, fmt.Errorf("Unknown model name: %s", name)
}

func getModelParams(name ModelName) (Params, error) {
	switch name {
	case ModelGPT4, ModelGPT35Turbo, ModelTextEmbeddingAda002:
		return getEncodingParams(EncodingCl100kBase)
	case ModelTextDavinci002, ModelTextDavinci003:
		return getEncodingParams(EncodingP50kBase)
	case ModelGPT2, ModelDavinci:
		return getEncodingParams(EncodingR50kBase)
	}

	return Params{}, fmt.Errorf("Unknown model name: %s", name)
}

func r50kBase() Params {
	// https://openaipublic.blob.core.windows.net/encodings/r50k_base.tiktoken
	mergeableRanks := loadTokenBytePairEncoding(r50kBaseLines)
	n := 50257

	return Params{
		ExplicitNVocab: &n,
		PatStr:         `'s|'t|'re|'ve|'m|'ll|'d| ?\p{L}+| ?\p{N}+| ?[^\s\p{L}\p{N}]+|\s+(?!\S)|\s+`,
		MergeableRanks: mergeableRanks,
		SpecialTokens: map[string]int{
			EndOfText: 50256,
		},
	}
}

func p50kBase() Params {
	// https://openaipublic.blob.core.windows.net/encodings/p50k_base.tiktoken
	mergeableRanks := loadTokenBytePairEncoding(p50kBaseLines)
	n := 50281

	return Params{
		ExplicitNVocab: &n,
		PatStr:         `'s|'t|'re|'ve|'m|'ll|'d| ?\p{L}+| ?\p{N}+| ?[^\s\p{L}\p{N}]+|\s+(?!\S)|\s+`,
		MergeableRanks: mergeableRanks,
		SpecialTokens: map[string]int{
			EndOfText: 50256,
		},
	}
}

func p50kEdit() Params {
	// https://openaipublic.blob.core.windows.net/encodings/p50k_base.tiktoken
	mergeableRanks := loadTokenBytePairEncoding(p50kBaseLines)

	return Params{
		PatStr:         `'s|'t|'re|'ve|'m|'ll|'d| ?\p{L}+| ?\p{N}+| ?[^\s\p{L}\p{N}]+|\s+(?!\S)|\s+`,
		MergeableRanks: mergeableRanks,
		SpecialTokens: map[string]int{
			EndOfText: 50256,
			FimPrefix: 50281,
			FimMiddle: 50282,
			FimSuffix: 50283,
		},
	}
}

func cl100kBase() Params {
	// https://openaipublic.blob.core.windows.net/encodings/cl100k_base.tiktoken
	mergeableRanks := loadTokenBytePairEncoding(cl100kBaseLines)

	return Params{
		PatStr:         `(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}{1,3}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+`,
		MergeableRanks: mergeableRanks,
		SpecialTokens: map[string]int{
			EndOfText:   100257,
			FimPrefix:   100258,
			FimMiddle:   100259,
			FimSuffix:   100260,
			EndOfPrompt: 100276,
		},
	}
}

func loadTokenBytePairEncoding(linesStr string) (result map[string]int) {
	result = map[string]int{}

	lines := strings.Split(linesStr, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		splitted := strings.Split(line, " ")
		if len(splitted) == 2 {
			k := splitted[0]
			if decoded, err := base64.StdEncoding.DecodeString(k); err == nil {
				k = string(decoded)
			} else {
				log.Printf("* could not decode base64 string: %s", splitted[0])
			}
			v := splitted[1]

			if i, err := strconv.Atoi(v); err == nil {
				result[k] = i
			}
		} else {
			log.Printf("* malformed line in data: %s", line)
		}
	}

	return result
}
