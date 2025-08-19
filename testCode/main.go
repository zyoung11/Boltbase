package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/imroc/req/v3"
)

// =============================
var WriteConcurrency int = 10
var SendSum int = 1000

var ReadConcurrency int = 10
var ReadSum int = 10000

//
//=============================

type ApiKeyResult struct {
	ApiKey string `json:"apiKey"`
}

var (
	apiKeyResult ApiKeyResult
	wg           sync.WaitGroup
)

func main() {
	fmt.Printf("\n写入并发：%v\n写入总数：%v\n读取并发：%v\n读取并发：%v\n", WriteConcurrency, SendSum, ReadConcurrency, ReadSum)
	client := req.C()
	setPassword(client)
	setApiKey(client)
	createBucket(client)
	writeTimer(WriteConcurrency, SendSum, "Seq", "test1", client, sent)
	writeTimer(WriteConcurrency, SendSum, "Time", "test2", client, sent)
	readTimer(ReadConcurrency, ReadSum, "Seq", map[string]string{"type": "get", "bucket": "test1", "Q": "1"}, client, read)
	readTimer(ReadConcurrency, ReadSum, "Seq", map[string]string{"type": "all", "bucket": "test1", "Q": ""}, client, read)
	readTimer(ReadConcurrency, ReadSum, "Time", map[string]string{"type": "all", "bucket": "test2", "Q": ""}, client, read)
	time.Sleep(1 * time.Second)
	removeFile("./Boltbase.db")
}

func readTimer(c, n int, name string, params map[string]string, client *req.Client, f func(int, int, map[string]string, *req.Client)) {
	start := time.Now()
	f(c, n, params, client)
	end := time.Since(start)
	endf := end.Seconds()
	QPS := float64(n) / endf
	fmt.Printf("\nRead\n%s time: %v\nQPS: %v\n", name, end, QPS)
}

func read(c, n int, params map[string]string, client *req.Client) {
	m := n / c
	for range c {
		wg.Add(1)
		go func() {
			for range int(m) {
				_, err := client.R().
					SetHeader("Authorization", apiKeyResult.ApiKey).
					SetPathParams(params).
					Get("http://localhost:5090/kv/{type}/{bucket}/{Q}")
				if err != nil {
					log.Fatal(err)
				}
			}
			defer wg.Done()
		}()
	}
	wg.Wait()
}

func writeTimer(c, n int, name, bucket string, client *req.Client, f func(int, int, string, *req.Client)) {
	start := time.Now()
	f(c, n, bucket, client)
	end := time.Since(start)
	endf := end.Seconds()
	QPS := float64(n) / endf
	fmt.Printf("\nWrite\n%s time: %v\nIPS: %v\n", name, end, QPS)
}

func removeFile(path string) {
	if err := os.Remove(path); err != nil {
		log.Fatal(err)
	}
}

func sent(c, n int, bucket string, client *req.Client) {
	type Body struct {
		Bucket string `json:"bucket"`
		Value  string `json:"value"`
	}
	m := n / c
	for range c {
		wg.Add(1)
		go func() {
			for range int(m) {
				resp, err := client.R().
					SetHeader("Authorization", apiKeyResult.ApiKey).
					SetBody(&Body{
						Bucket: bucket,
						Value:  "qwertyuiopasdfghjklzxcvbnm1234567890",
					}).
					Post("http://localhost:5090/kv")
				if err != nil {
					log.Fatal(err)
				}
				if !resp.IsSuccessState() {
					fmt.Println("bad response status:", resp.Status)
				}
			}
			defer wg.Done()
		}()
	}
	wg.Wait()
}

func createBucket(client *req.Client) {
	resp1, err := client.R().
		SetHeader("Authorization", apiKeyResult.ApiKey).
		Post("http://localhost:5090/bucket/test1/seq")
	if err != nil {
		log.Fatal(err)
	}
	if !resp1.IsSuccessState() {
		fmt.Println("bad response status:", resp1.Status)
	}
	resp2, err := client.R().
		SetHeader("Authorization", apiKeyResult.ApiKey).
		Post("http://localhost:5090/bucket/test2/time")
	if err != nil {
		log.Fatal(err)
	}
	if !resp2.IsSuccessState() {
		fmt.Println("bad response status:", resp2.Status)
	}

}

func setPassword(client *req.Client) {
	type Password struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	resp, err := client.R().SetBody(&Password{
		Username: "zy",
		Password: "123",
	}).Post("http://localhost:5090/auth/password")
	if err != nil {
		log.Fatal(err)
	}
	if !resp.IsSuccessState() {
		fmt.Println("bad response status:", resp.Status)
	}
}

func setApiKey(client *req.Client) {
	type ApiKey struct {
		Duration string `json:"duration"`
	}
	resp, err := client.R().
		SetBasicAuth("zy", "123").
		SetBody(&ApiKey{
			Duration: "365d",
		}).
		SetSuccessResult(&apiKeyResult).
		Post("http://localhost:5090/auth/apikey")
	if err != nil {
		log.Fatal(err)
	}
	if !resp.IsSuccessState() {
		fmt.Println("bad response status:", resp.Status)
	}
	fmt.Printf("\nAPI Key: %s\n", apiKeyResult.ApiKey)
}
