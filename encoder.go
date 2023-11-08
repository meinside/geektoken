package geektoken

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	pcre "github.com/GRbit/go-pcre"
)

type EncoderType map[string]int

func (e *EncoderType) tryGetValue(key []byte) (result int, exists bool) {
	result, exists = (*e)[string(key)]
	return result, exists
}

type DecoderType map[int]string

func (d *DecoderType) tryGetValue(key int) ([]byte, bool) {
	result, exists := (*d)[key]
	if exists {
		return []byte(result), exists
	} else {
		return nil, exists
	}
}

// BPECore struct
type BPECore struct {
	Encoder                  EncoderType
	SpecialTokensEncoder     EncoderType
	Decoder                  DecoderType
	SpecialTokensDecoder     DecoderType
	RegexTls                 pcre.Regexp
	SpecialTokenPatternRegex pcre.Regexp
}

// returns a new BPECore
func newBPECore(bytePairEncoder EncoderType, specialTokenEncoder EncoderType, tokenPatternRegex *pcre.Regexp) BPECore {
	var _encoder EncoderType
	if bytePairEncoder == nil {
		_encoder = EncoderType{}
	} else {
		_encoder = bytePairEncoder
	}

	_decoder := DecoderType{}
	for k, v := range bytePairEncoder {
		_decoder[v] = k
	}

	var _specialTokenEncoder EncoderType
	if specialTokenEncoder == nil {
		_specialTokenEncoder = EncoderType{}
	} else {
		_specialTokenEncoder = specialTokenEncoder
	}
	_specialTokenDecoder := DecoderType{}
	if specialTokenEncoder == nil {
		_specialTokenDecoder = DecoderType{}
	} else {
		for k, v := range specialTokenEncoder {
			_specialTokenDecoder[v] = k
		}
	}

	var _regexTLS, _specialTokenPatternRegex pcre.Regexp
	if tokenPatternRegex == nil {
		_regexTLS = pcre.MustCompileParse("")
	} else {
		_regexTLS = *tokenPatternRegex

		keys := []string{}
		for k := range _specialTokenEncoder {
			keys = append(keys, regexp.QuoteMeta(k))
		}
		_specialTokenPatternRegex = pcre.MustCompileParse(strings.Join(keys, "|"))
	}

	return BPECore{
		Encoder:                  _encoder,
		SpecialTokensEncoder:     _specialTokenEncoder,
		Decoder:                  _decoder,
		SpecialTokensDecoder:     _specialTokenDecoder,
		RegexTls:                 _regexTLS,
		SpecialTokenPatternRegex: _specialTokenPatternRegex,
	}
}

func (c *BPECore) encodeNative(text string, allowedSpecial map[string]bool) (result []int, count int) {
	encodedTokens := []int{}
	startIndex := 0
	lastPieceTokenLength := 0

	for {
		nextSpecialStartIndex := findNextSpecialStartIndex(text, allowedSpecial, startIndex, c.SpecialTokenPatternRegex)

		var endIndex int
		if nextSpecialStartIndex != nil {
			endIndex = *nextSpecialStartIndex
		} else {
			endIndex = len(text)
		}
		textSegment := text[startIndex:endIndex]

		matcher := c.RegexTls.NewMatcherString(textSegment, 0)
		if matcher.Matches {
			for _, match := range matcher.ExtractString() {
				encodedPiece := []byte(match)

				if token, success := c.Encoder.tryGetValue(encodedPiece); success {
					lastPieceTokenLength = 1
					encodedTokens = append(encodedTokens, token)
					continue
				}

				tokens := bytePairEncode(encodedPiece, c.Encoder)

				lastPieceTokenLength = len(tokens)
				encodedTokens = append(encodedTokens, tokens...)
			}
		}

		if nextSpecialStartIndex != nil {
			specialToken := text[*nextSpecialStartIndex:]
			specialTokenValue := c.SpecialTokensEncoder[specialToken]
			encodedTokens = append(encodedTokens, specialTokenValue)
			startIndex = *nextSpecialStartIndex + len(specialToken)
			lastPieceTokenLength = 0
		} else {
			break
		}
	}

	return encodedTokens, lastPieceTokenLength
}

func findNextSpecialStartIndex(text string, allowedSpecial map[string]bool, startIndex int, specialRegex pcre.Regexp) *int {
	searchIndex := startIndex

	for {
		searched := text[searchIndex:]
		nextSpecialMatch := specialRegex.FindIndex([]byte(searched), 0)
		if nextSpecialMatch == nil {
			return nil
		}

		specialToken := searched[nextSpecialMatch[0]:nextSpecialMatch[1]]

		if v, exists := allowedSpecial[specialToken]; exists && v {
			result := nextSpecialMatch[0] + searchIndex
			return &result
		}

		searchIndex = nextSpecialMatch[0] + searchIndex + 1
	}
}

func (c *BPECore) decodeNative(tokens []int) []byte {
	decodedBytes := []byte{}
	for _, token := range tokens {
		var tokenBytes []byte
		var success bool
		if tokenBytes, success = c.tryDecodeToken(token); !success {
			continue
		}

		if tokenBytes != nil {
			decodedBytes = append(decodedBytes, tokenBytes...)
		}
	}

	return decodedBytes
}

func (c *BPECore) tryDecodeToken(token int) (result []byte, success bool) {
	if result, success = c.Decoder.tryGetValue(token); !success {
		result, success = c.SpecialTokensDecoder.tryGetValue(token)
	}
	return result, success
}

func bytePairMerge[T any](piece []byte, ranks EncoderType, f func(start, end int) T) []T {
	type partition struct {
		start int
		rank  int
	}

	var partitions = []partition{}
	for i := 0; i <= len(piece); i++ {
		partitions = append(partitions, partition{start: i, rank: math.MaxInt})
	}

	getRank := func(partitions []partition, startIndex, skip int) *int {
		if startIndex+skip+2 >= len(partitions) {
			return nil
		}

		key := piece[partitions[startIndex].start:partitions[startIndex+skip+2].start]

		if rank, success := ranks.tryGetValue(key); success {
			return &rank
		} else {
			return nil
		}
	}

	for i := 0; i < len(partitions)-2; i++ {
		if rank := getRank(partitions, i, 0); rank != nil {
			partitions[i].rank = *rank
		}
	}

	type rankIndex struct {
		rank  int
		index int
	}

	for {
		if len(partitions) == 1 {
			break
		}

		minRank := rankIndex{rank: math.MaxInt, index: 0}
		for i, part := range partitions[:len(partitions)-1] {
			if part.rank < minRank.rank {
				minRank = rankIndex{rank: part.rank, index: i}
			}
		}

		if minRank.rank != math.MaxInt {
			i := minRank.index

			var rank int
			if rankPtr := getRank(partitions, i, 1); rankPtr != nil {
				rank = *rankPtr
			} else {
				rank = math.MaxInt
			}
			partitions[i].rank = rank

			if i > 0 {
				if rankPtr := getRank(partitions, i-1, 1); rankPtr != nil {
					rank = *rankPtr
				} else {
					rank = math.MaxInt
				}
				partitions[i-1].rank = rank
			}

			// remove partition at index: minRank.Index + 1
			partitions = append(partitions[:i+1], partitions[i+2:]...)
		} else {
			break
		}
	}

	output := []T{}
	for i := 0; i < len(partitions)-1; i++ {
		output = append(output, f(partitions[i].start, partitions[i+1].start))
	}

	return output
}

func bytePairEncode(piece []byte, ranks EncoderType) []int {
	if len(piece) == 1 {
		return []int{ranks[string(piece)]}
	}

	return bytePairMerge(piece, ranks, func(start, end int) int {
		key := piece[start:end]
		return ranks[string(key)]
	})
}

func bytePairSplit(piece []byte, ranks EncoderType) [][]byte {
	if len(piece) == 1 {
		return [][]byte{piece}
	}

	return bytePairMerge(piece, ranks, func(start, end int) []byte {
		return piece[start:end]
	})
}

// get max value from given EncoderType `dic`
func maxValue(dic EncoderType) (max int) {
	max = 0

	for _, v := range dic {
		if v > max {
			max = v
		}
	}

	return max
}

// get max value between `a` and `b`
func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

// Tokenizer struct
type Tokenizer struct {
	bytePairEncodingCoreProcessor BPECore
	specialTokenMappings          EncoderType
	maxTokenValue                 int
}

// GetTokenizerWithEncoding returns a Tokenizer with given encoding name
func GetTokenizerWithEncoding(encoding EncodingName) (result Tokenizer, err error) {
	var params Params
	if params, err = getEncodingParams(encoding); err == nil {
		result, err = newTokenizer(params.PatStr, params.MergeableRanks, params.SpecialTokens, params.ExplicitNVocab)
	}

	return result, err
}

func GetTokenizerWithModel(model ModelName) (result Tokenizer, err error) {
	var params Params
	if params, err = getModelParams(model); err == nil {
		result, err = newTokenizer(params.PatStr, params.MergeableRanks, params.SpecialTokens, params.ExplicitNVocab)
	}

	return result, err
}

// returns a new Tokenizer
func newTokenizer(pattern string, bytePairRanks EncoderType, specialTokenMappings EncoderType, explicitNVocab *int) (Tokenizer, error) {
	maxTokenValue := max(maxValue(bytePairRanks), maxValue(specialTokenMappings))

	if explicitNVocab != nil {
		if len(bytePairRanks)+len(specialTokenMappings) != *explicitNVocab {
			return Tokenizer{}, fmt.Errorf("The number of mergeable tokens and special tokens must be equal to explicit_n_vocab.")
		}

		if maxTokenValue != *explicitNVocab-1 {
			return Tokenizer{}, fmt.Errorf("The maximum token value must be equal to explicit_n_vocab - 1.")
		}
	}

	regex := pcre.MustCompileParse(pattern)

	return Tokenizer{
		bytePairEncodingCoreProcessor: newBPECore(bytePairRanks, specialTokenMappings, &regex),
		specialTokenMappings:          specialTokenMappings,
		maxTokenValue:                 maxTokenValue,
	}, nil
}

func specialTokenRegex(tokens []string) pcre.Regexp {
	escaped := []string{}
	for _, token := range tokens {
		escaped = append(escaped, regexp.QuoteMeta(token))
	}
	pattern := fmt.Sprintf("(%s)", strings.Join(escaped, "|"))
	return pcre.MustCompileParse(pattern)
}

// Encode encodes given `lineToEncode` string with allowed/disallowed specials
func (e *Tokenizer) Encode(lineToEncode string, allowedSpecial []string, disallowedSpecial []string) (result []int, err error) {
	specialTokenSet := map[string]bool{}
	for k := range e.specialTokenMappings {
		specialTokenSet[k] = true
	}

	allowed := map[string]bool{}
	for _, v := range allowedSpecial {
		allowed[v] = true
	}

	disallowed := map[string]bool{}
	if disallowedSpecial != nil {
		for _, v := range disallowedSpecial {
			disallowed[v] = true
		}
	} else {
		disallowed = map[string]bool{"all": true}
	}

	if v, exists := disallowed["all"]; exists && v {
		// copy disallowed <- specialTokenSet
		disallowed = map[string]bool{}
		for k := range specialTokenSet {
			disallowed[k] = true
		}

		// except allowed ones
		for k := range allowed {
			delete(disallowed, k)
		}
	}

	if v, exists := allowed["all"]; exists && v {
		allowed = specialTokenSet
	}

	if len(disallowed) > 0 {
		disallowedTokens := []string{}
		for k := range disallowed {
			disallowedTokens = append(disallowedTokens, k)
		}
		regexPattern := specialTokenRegex(disallowedTokens)

		if match := regexPattern.FindIndex([]byte(lineToEncode), 0); match != nil {
			return nil, fmt.Errorf("Disallowed special token found: %s", lineToEncode[match[0]:match[1]])
		}
	}

	encodedLine, _ := e.bytePairEncodingCoreProcessor.encodeNative(lineToEncode, allowed)

	return encodedLine, nil
}

// Decode decodes given `inputTokensToDecode`
func (e *Tokenizer) Decode(inputTokensToDecode []int) string {
	bytes := e.bytePairEncodingCoreProcessor.decodeNative(inputTokensToDecode)
	return string(bytes)
}
