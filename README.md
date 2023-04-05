# geektoken

A [BPE](https://en.wikipedia.org/wiki/Byte_pair_encoding) tokenizer for use with OpenAI's models,

ported and referenced from [tiktoken](https://github.com/openai/tiktoken) and [SharpToken](https://github.com/dmitry-brazhenko/SharpToken).

## requirements

Go standard library doesn't support [PCRE](https://www.pcre.org/), so it depends on [go-pcre](https://github.com/GRbit/go-pcre).

It requires `libpcre3-dev` or `libpcre++-dev` to be installed on the system.

## usage

```go
package main

import (
    "log"

    "github.com/meinside/geektoken"
)

func main() {
    //text := "Hellow, world!"
    text := "나는 우리나라가 세계에서 가장 아름다운 나라가 되기를 원한다. 가장 부강한 나라가 되기를 원하지 않는다."

    tokenizer, _ := geektoken.GetTokenizerWithModel(geektoken.ModelGPT35Turbo)
    if encoded, err := tokenizer.Encode(text, nil, nil); err == nil {
        log.Printf("encoded token: %+v, token count = %d", encoded, len(encoded))
    }
}
```

## known issues / todos

- [ ] Some encoded bytes differ from the ones from other BPE libraries
- [ ] Add more tests
- [ ] Optimize codes

## license

MIT

