package profanity

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestFilter(t *testing.T) {
	words := strings.Split(InvalidWords, ",")
	for _, v := range words {
		InvalidWord[v] = nil
	}
	var Set = make(map[string]interface{})
	Set["你妈逼的"] = nil
	Set["你妈"] = nil
	Set["狗日"] = nil
	Set["傻"] = nil
	Set["fuck"] = nil
	AddSensitiveToMap(Set)
	text := "文明用语你&* 妈, 逼的你这个狗 日的，怎么这么傻啊。我也f u@c%k是服了，我日,这些话我都说不出口"
	ret := ChangeSensitiveWords(text)
	except := "文明用语*****, 逼的你这个***的，怎么这么*啊。我也*******是服了，我日,这些话我都说不出口"
	in := []rune(text)
	out := []rune(ret)
	exceptR := []rune(except)
	assert.Equal(t, except, ret)
	assert.Equal(t, len(exceptR), len(in), len(out))
}
