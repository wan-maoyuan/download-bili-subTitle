package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

//用来获取B站视频标题的URL
const TitleUrl = "https://api.bilibili.com/x/web-interface/view?bvid="

//换行符
var Sep = "\r\n"

type videoInfo struct {
	id          string //视频ID
	url         string //视频的链接
	subTitleUrl string //视频字幕链接
	title       string //视频的标题
}

//bilibili视频的信息
type bilibiliInfo struct {
	Data struct {
		Title string `json:"title"`
	} `json:"data"`
}

func main() {
	urlList := readUrlFile()
	for _, url := range urlList {
		video, err := fillingVideoInfo(url)
		if err == nil {
			getSubTitleAndSave(video)
		}
	}
	fmt.Println("字幕文件全部下载成功")
	time.Sleep(time.Second * 5)
}

//程序初始化
func init() {
	//判断是否存在 字幕 文件夹，如果存在，删除并创建一个新的，如果不存在直接创建一个新的
	_, err := os.Stat("./字幕")
	if err == nil {
		err := os.RemoveAll("./字幕") //删除字幕文件夹以及里面所有的文件
		if err != nil {
			panic("字幕文件夹删除失败")
		}
	}
	err = os.Mkdir("字幕", fs.ModeDir)
	if err != nil {
		panic("字幕文件夹创建失败：" + err.Error())
	}
}

//填充B站视频的信息
func fillingVideoInfo(url string) (videoInfo, error) {
	var video videoInfo
	array := strings.Split(url, "/")
	if len(array) < 2 {
		return videoInfo{}, errors.New("链接不是B站视频")
	}
	video.id = array[len(array)-1]
	video.url = url
	newUrl, err := formatUrl(url)
	if err != nil {
		return videoInfo{}, err
	}
	video.subTitleUrl = newUrl
	video.title, err = getVideoTitle(video.id)
	if err != nil {
		return videoInfo{}, err
	}
	return video, nil
}

//读取存放B站视频链接文件, 返回一个存放URL的数组, 过滤掉空的URL
func readUrlFile() []string {
	fmt.Println("开始读取 链接.txt 文件")
	file, err := os.Open("链接.txt")
	if err != nil {
		panic("没有找到 链接.txt 文件")
	}
	urlContext := bytes.NewBuffer(nil)

	defer func() {
		err := file.Close()
		if err != nil {
			panic("链接.txt 关闭失败")
		}
	}()

	var buffer [512]byte
	for {
		n, err := file.Read(buffer[0:])
		urlContext.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			panic("读取 链接.txt 文件失败 ")
		}
	}

	var urlList []string
	for _, url := range strings.Split(urlContext.String(), Sep) {
		if url != "" {
			urlList = append(urlList, url)
			fmt.Println(url)
		}
	}
	fmt.Println("链接.txt 文件 读取完成")
	return urlList
}

//获取B站视频的标题
func getVideoTitle(id string) (string, error) {
	url := TitleUrl + id
	client := &http.Client{Timeout: 10 * time.Second} //超时时间设置为10s
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
	}

	var bilibili bilibiliInfo
	if err := json.Unmarshal([]byte(result.String()), &bilibili); err == nil {
		return bilibili.Data.Title, nil
	} else {
		return "", err
	}
}

//格式化URL
//输入：https://www.bilibili.com/video/BV1VM4y1T7Kw
//输出：https://www.bilibili-bb.com/video/BV1VM4y1T7Kw
func formatUrl(url string) (string, error) {
	var newUrl string
	array := strings.Split(url, ".")
	for index, _ := range array {
		if array[index] == "bilibili" {
			array[index] += "-bb"
		}
	}
	newUrl = strings.Join(array, ".")
	if len(url) == len(newUrl) {
		fmt.Println("链接不是B站视频")
		return "", errors.New("链接不是B站视频")
	}
	return newUrl, nil
}

//获取视频的字幕信息，并以标题为文件名保存为文件
func getSubTitleAndSave(video videoInfo) {
	fmt.Println("开始下载并写入", video.title)
	subTitle, err := getSubTitle(video.subTitleUrl)
	if err != nil {
		return
	}
	if strings.Count(subTitle, "") <= 150 {
		fmt.Println(video.title, "获取字幕失败")
		return
	}
	subTitle = subTitle[148:]
	fileName := "./字幕/" + video.title + ".txt"
	file, openErr := os.Create(fileName)
	if openErr != nil {
		fmt.Println("文件打开失败")
		return
	}
	_, err = file.Write([]byte(subTitle))
	if err != nil {
		fmt.Println("文件写入失败")
		return
	}
	fmt.Println("下载成功，完成写入")
}

//根据URL获取视频的字幕
func getSubTitle(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
	}

	return result.String(), nil
}
