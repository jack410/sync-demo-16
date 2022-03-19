package main

import (
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

//下面这句话的意思是打包go的时候把后面这个目录打包进去
//go:embed frontend/dist/*
var FS embed.FS

func main() {
	go func() {
		gin.SetMode(gin.DebugMode)
		router := gin.Default()
		//把打包好的静态文件变成一个结构化的目录
		staticFiles, _ := fs.Sub(FS, "frontend/dist")
		router.StaticFS("/static", http.FS(staticFiles))
		//NoRoute表示用户访问路径没匹配到程序定义的路由
		router.NoRoute(func(c *gin.Context) {
			//获取用户访问的路径
			path := c.Request.URL.Path
			//判断路径是否以static开头
			if strings.HasPrefix(path, "/static/") {
				reader, err := staticFiles.Open("index.html")
				if err != nil {
					log.Fatal(err)
				}
				defer reader.Close()
				stat, err := reader.Stat()
				if err != nil {
					log.Fatal(err)
				}
				c.DataFromReader(http.StatusOK, stat.Size(), "text/html", reader, nil)
				//如果不是static开头则返回404
			} else {
				c.Status(http.StatusNotFound)
			}
		})
		router.Run(":8080")
	}()
	chSignal := make(chan os.Signal, 1)
	//signal.Notify订阅os.Interrupt信号,一旦有信号则往chSingal管道里写一个信号
	signal.Notify(chSignal, os.Interrupt)

	chromePath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	var tmpDir string
	name, _ := ioutil.TempDir("", "lorca")
	tmpDir = name
	//删除缓存文件
	defer os.RemoveAll(tmpDir)
	fmt.Println(tmpDir)
	cmd := exec.Command(chromePath, "--app=http://localhost:8080/static/index.html", fmt.Sprintf("--user-data-dir=%s", tmpDir),
		"--no-first-run")
	cmd.Start()
	//如果没有值则一直等待（阻塞），直到有信号输入
	//select可以监听多个管道，只要有一个管道有信号则进行下一步
	//如果没有信号，select就等待（阻塞）
	select {
	case <-chSignal:
		//一旦有信号则关闭浏览器
		cmd.Process.Kill()
	}
}
