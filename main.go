package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/gin-gonic/gin"
)

type File struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Payload struct {
	Cmd         string   `json:"cmd"`
	Args        []string `json:"args"`
	Async       bool     `json:"async"`
	CallbackUrl string   `json:"callbackUrl"`
	Files       []File   `json:"files"`
}

type AsyncResult struct {
	Success bool   `json:"success"`
	JobId   int    `json:"jobId"`
	Result  string `json:"result"`
}

var TMP_DIR = "./tmp"

func runCode(jobId int, payload Payload) string {
	wd := path.Join(TMP_DIR, strconv.Itoa(jobId))
	os.Mkdir(wd, os.ModePerm)

	for _, file := range payload.Files {
		content := []byte(file.Content)
		err := os.WriteFile(path.Join(wd, file.Name), content, os.ModePerm)
		if err != nil {
			log.Fatalln(err)
		}
	}

	cmd := exec.Command(payload.Cmd, payload.Args...)

	cmd.Dir = wd
	data, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}
	return string(data)
}

func runAndPost(jobId int, payload Payload) {
	result := runCode(jobId, payload)
	asyncResult := &AsyncResult{
		Success: true,
		JobId:   jobId,
		Result:  result,
	}

	jsonStr, errMarshal := json.Marshal(asyncResult)
	if errMarshal != nil {
		log.Fatalln(errMarshal)
	}
	req, errNewRequest := http.NewRequest("POST", payload.CallbackUrl, bytes.NewBuffer(jsonStr))
	if errNewRequest != nil {
		log.Fatalln(errNewRequest)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, errDoRequest := client.Do(req)
	if errDoRequest != nil {
		panic(errDoRequest)
	}
	defer resp.Body.Close()
}

func runCodeAuto(jobId int, payload Payload) string {
	if payload.Async {
		go runAndPost(jobId, payload)
		return "async running"
	} else {
		return runCode(jobId, payload)
	}
}

func main() {
	var jobId = 0
	r := gin.Default()

	r.POST("/echo", func(c *gin.Context) {
		var result AsyncResult
		if c.ShouldBind(&result) == nil {
			log.Println(result)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
			})
		}
	})

	r.POST("/run", func(c *gin.Context) {
		var payload Payload
		if c.ShouldBind(&payload) == nil {
			log.Println(payload)
			jobId++
			result := runCodeAuto(jobId, payload)

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"jobId":   jobId,
				"result":  result,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
			})
		}
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
