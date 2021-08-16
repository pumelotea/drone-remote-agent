package main

import (
	"io/ioutil"
	"os"
	"strings"
)

func ReadAll(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func NotEmptyCopy(dst string, src string) string {
	if src == "" {
		return dst
	}
	return src
}

func CombineScriptIntoOneLine(script string) string {
	if scripts == "" {
		return ""
	}
	oneline := ""
	list := strings.Split(script, ";")
	for i := 0; i < len(list); i++ {
		trimed := strings.TrimRight(list[i], " ")
		if trimed[len(trimed)-1:] == "\\" {
			oneline += trimed[0 : len(trimed)-1]
		} else {
			oneline += trimed + ";"
		}
	}
	return oneline
}
