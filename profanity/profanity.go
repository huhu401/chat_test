package profanity

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var sensitiveWord = make(map[string]interface{})
var InvalidWord = make(map[string]interface{}) //无效词汇，不参与敏感词汇判断直接忽略

const InvalidWords = " ,~,!,@,#,$,%,^,&,*,(,),_,-,+,=,?,<,>,.,—,，,。,/,\\,|,《,》,？,;,:,：,',‘,；,“,"

func BeforeMain() {
	load()
}

func load() {
	fp, err := os.Open("profanity.log")
	if err != nil {
		fmt.Println(err) //打开文件错误
		return
	}
	words := strings.Split(InvalidWords, ",")
	for _, v := range words {
		InvalidWord[v] = nil
	}
	buf := bufio.NewScanner(fp)
	var Set = make(map[string]interface{})
	for {
		if !buf.Scan() {
			break //文件读完了,退出for
		}
		line := buf.Text() //获取每一行
		Set[line] = nil
	}
	AddSensitiveToMap(Set)
}

//生成违禁词集合
func AddSensitiveToMap(set map[string]interface{}) {
	for key := range set {
		str := []rune(key)
		nowMap := sensitiveWord
		for i := 0; i < len(str); i++ {
			if _, ok := nowMap[string(str[i])]; !ok { //如果该key不存在，
				thisMap := make(map[string]interface{})
				thisMap["isEnd"] = false
				nowMap[string(str[i])] = thisMap
				nowMap = thisMap
			} else {
				nowMap = nowMap[string(str[i])].(map[string]interface{})
			}
			if i == len(str)-1 {
				nowMap["isEnd"] = true
			}
		}

	}
}

//敏感词汇转换为*
func ChangeSensitiveWords(txt string) (word string) {
	str := []rune(txt)
	nowMap := sensitiveWord
	start := -1
	tag := -1
	for i := 0; i < len(str); i++ {
		if _, ok := InvalidWord[(string(str[i]))]; ok {
			continue //如果是无效词汇直接跳过
		}
		if thisMap, ok := nowMap[string(str[i])].(map[string]interface{}); ok {
			//记录敏感词第一个文字的位置
			tag++
			if tag == 0 {
				start = i

			}
			//判断是否为敏感词的最后一个文字
			if isEnd, _ := thisMap["isEnd"].(bool); isEnd {
				//将敏感词的第一个文字到最后一个文字全部替换为“*”
				for y := start; y < i+1; y++ {
					str[y] = 42
				}
				//重置标志数据
				nowMap = sensitiveWord
				start = -1
				tag = -1

			} else { //不是最后一个，则将其包含的map赋值给nowMap
				nowMap = nowMap[string(str[i])].(map[string]interface{})
			}

		} else { //如果敏感词不是全匹配，则终止此敏感词查找。从开始位置的第二个文字继续判断
			if start != -1 {
				i = start + 1
			}
			//重置标志参数
			nowMap = sensitiveWord
			start = -1
			tag = -1
		}
	}

	return string(str)
}
