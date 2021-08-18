package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"
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

func bar(count, size int) string {
	str := ""
	for i := 0; i < size; i++ {
		if i < count {
			str += "="
		} else {
			str += " "
		}
	}
	return str
}

// PKCS7Padding PKCS7 填充模式
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	//Repeat()函数的功能是把切片[]byte{byte(padding)}复制padding个，然后合并成新的字节切片返回
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS7UnPadding 填充的反向操作，删除填充字符串
func PKCS7UnPadding(origData []byte) ([]byte, error) {
	//获取数据长度
	length := len(origData)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	} else {
		//获取填充字符串长度
		unpadding := int(origData[length-1])
		//截取切片，删除填充字节，并且返回明文
		return origData[:(length - unpadding)], nil
	}
}

// AesEcrypt 实现加密
func AesEcrypt(origData []byte, key []byte) ([]byte, error) {
	//创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	//对数据进行填充，让数据长度满足需求
	origData = PKCS7Padding(origData, blockSize)
	//采用AES加密方法中CBC加密模式
	blocMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	//执行加密
	blocMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

// AesDeCrypt 实现解密
func AesDeCrypt(cypted []byte, key []byte) ([]byte, error) {
	//创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块大小
	blockSize := block.BlockSize()
	//创建加密客户端实例
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(cypted))
	//这个函数也可以用来解密
	blockMode.CryptBlocks(origData, cypted)
	//去除填充字符串
	origData, err = PKCS7UnPadding(origData)
	if err != nil {
		return nil, err
	}
	return origData, err
}

// PackBytes 封包二进制
func PackBytes(pwd []byte, data []byte) []byte {
	l := len(pwd)
	var result = make([]byte, 0)
	result = append(result, IntToBytes(l)...)
	result = append(result, pwd...)
	result = append(result, data...)
	return result
}

// UnPackBytes 二进制解包
func UnPackBytes(data []byte) (*PackData, error) {
	//首字节 4位为解密秘钥长度
	if len(data) < 4 {
		return nil, errors.New("byte len is too short")
	}
	l := BytesToInt(data[0:4])
	pwd := data[4 : l+4]
	d := data[l+4:]
	return &PackData{
		Len:  l,
		Pwd:  pwd,
		Data: d,
	}, nil
}

func GenerateAESPwd() []byte {
	var LetterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	const AesPwdLen = 16
	b := make([]rune, AesPwdLen)
	for i := range b {
		b[i] = LetterRunes[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(LetterRunes))]
	}
	return []byte(string(b))
}

func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}
