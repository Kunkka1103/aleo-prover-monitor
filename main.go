package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"aleo-prover-monitor/prometh"
)

var apiBaseURL = flag.String("api", "http://localhost:8088", "Base URL of the API")
var pushGatewayAddr = flag.String("pushGateway", "http://pushgateway:9091", "pushgateway addr")
var interval = flag.Int("interval", 5, "check interval(min)")
var addressFile = flag.String("addrFile", "", "addressFile")

type SpeedRequestPayload struct {
	Address  []string `json:"address"`
	Duration int      `json:"duration"`
}

type SpeedResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []struct {
			Address string `json:"address"`
			Speed   string `json:"speed"`
		} `json:"list"`
		Total string `json:"total"`
	} `json:"data"`
}

type RewardRequestPayload struct {
	Address []string `json:"address"`
}

type RewardResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []struct {
			Address     string `json:"address"`
			TotalReward string `json:"total_reward"`
		} `json:"list"`
		Total string `json:"total"`
	} `json:"data"`
}

type HeightRequestPayload struct {
	Address []string `json:"address"`
}

type HeightResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		Address string `json:"address"`
		Height  int    `json:"height"`
	} `json:"data"`
}

type BlockData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Height         int    `json:"height"`
		ProofTarget    string `json:"proof_target"`
		CoinbaseReward string `json:"coinbase_reward"`
	} `json:"data"`
}

func main() {
	flag.Parse()
	addresses, err := readAddressesFromFile(*addressFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	duration := []int{900, 3600, 43200, 86400}

	for {
		//Speed
		SpeedURL := *apiBaseURL + "/api/v1/provers/prover_speed_list"
		for _, d := range duration {
			speedRespon, err := SpeedSendRequest(SpeedURL, SpeedRequestPayload{addresses, d})
			if err != nil {
				log.Printf("%s 请求失败:%s\n", SpeedURL, err)
				time.Sleep(time.Duration(*interval) * time.Minute)
				continue
			}
			log.Printf("%s 请求成功\n", SpeedURL)

			for _, r := range speedRespon.Data.List {
				prometh.SpeedPush(*pushGatewayAddr, r.Address, d, r.Speed)
			}
			prometh.TotalSpeedPush(*pushGatewayAddr, d, speedRespon.Data.Total)
		}

		//Reward
		RewardURL := *apiBaseURL + "/api/v1/provers/prover_reward_list"
		rewardRespon, err := RewardSendRequest(RewardURL, RewardRequestPayload{addresses})
		if err != nil {
			log.Printf("%s 请求失败:%s", RewardURL, err)
			time.Sleep(time.Duration(*interval) * time.Minute)
			continue
		}

		for _, r := range rewardRespon.Data.List {
			prometh.RewardPush(*pushGatewayAddr, r.Address, r.TotalReward)
		}
		prometh.TotalRewardPush(*pushGatewayAddr, rewardRespon.Data.Total)

		//Height
		HeightURL := *apiBaseURL + "/api/v1/provers/prover_latest_height"
		heightRespon, err := HeightSendRequest(HeightURL, HeightRequestPayload{addresses})
		if err != nil {
			log.Printf("%s 请求失败:%s", HeightURL, err)
			time.Sleep(time.Duration(*interval) * time.Minute)
			continue
		}
		log.Printf("%s 请求成功\n", HeightURL)

		for _, r := range heightRespon.Data {
			prometh.HeightPush(*pushGatewayAddr, r.Address, r.Height)
		}

		//block
		BlockURL := *apiBaseURL + "/api/v1/chain/latest_block"
		blockRespon, err := BlockSendRequest(BlockURL)
		if err != nil {
			log.Printf("%s 请求失败:%s", BlockURL, err)
			time.Sleep(time.Duration(*interval) * time.Minute)
			continue
		}
		log.Printf("%s 请求成功\n", BlockURL)

		prometh.BlockPush(*pushGatewayAddr, blockRespon.Data.Height, blockRespon.Data.ProofTarget, blockRespon.Data.CoinbaseReward)

		//Sleep

		time.Sleep(time.Duration(*interval) * time.Minute)
	}

}

func SpeedSendRequest(url string, payload SpeedRequestPayload) (SpeedResponse, error) {
	var response SpeedResponse

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return response, fmt.Errorf("JSON序列化错误: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return response, fmt.Errorf("创建请求错误: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("发送请求错误: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("读取响应错误: %v", err)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, fmt.Errorf("JSON反序列化错误: %v", err)
	}

	return response, nil
}

func RewardSendRequest(url string, payload RewardRequestPayload) (RewardResponse, error) {
	var response RewardResponse

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return response, fmt.Errorf("JSON序列化错误: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return response, fmt.Errorf("创建请求错误: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("发送请求错误: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("读取响应错误: %v", err)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, fmt.Errorf("JSON反序列化错误: %v", err)
	}

	return response, nil
}

func HeightSendRequest(url string, payload HeightRequestPayload) (HeightResponse, error) {
	var response HeightResponse

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return response, fmt.Errorf("JSON序列化错误: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return response, fmt.Errorf("创建请求错误: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("发送请求错误: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("读取响应错误: %v", err)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, fmt.Errorf("JSON反序列化错误: %v", err)
	}

	return response, nil
}

func BlockSendRequest(url string) (BlockData, error) {
	var response BlockData

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return response, fmt.Errorf("创建请求错误: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("发送请求错误: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("读取响应错误: %v", err)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, fmt.Errorf("JSON反序列化错误: %v", err)
	}

	return response, nil
}

func readAddressesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("打开文件错误: %v", err)
	}
	defer file.Close()

	var addresses []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addresses = append(addresses, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件错误: %v", err)
	}

	return addresses, nil
}
