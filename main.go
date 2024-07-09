package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/pflag"
)

var (
	pushgatewayURL  string
	apiBaseURL      string
	addresses       []string
	defaultDuration = 5 * time.Minute
)

func main() {
	pflag.StringVar(&pushgatewayURL, "pushgateway", "http://pushgateway:9091", "URL of the Pushgateway")
	pflag.StringVar(&apiBaseURL, "api", "http://localhost:8088", "Base URL of the API")
	pflag.StringArrayVar(&addresses, "addresses", []string{}, "Addresses to monitor")

	pflag.DurationVar(&defaultDuration, "interval", 5*time.Minute, "Interval for fetching data")
	pflag.Parse()

	if len(addresses) == 0 {
		log.Fatal("No addresses provided")
	}

	log.Println("Starting prover monitor")

	fetchDataAndPush() // 立即开始获取数据

	ticker := time.NewTicker(defaultDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fetchDataAndPush()
		}
	}
}

func fetchDataAndPush() {
	log.Println("Fetching data and pushing to Pushgateway")

	durations := map[string]int{
		"15m": 900,
		"1h":  3600,
		"12h": 43200,
		"24h": 86400,
	}

	for _, address := range addresses {
		for durationName, duration := range durations {
			log.Printf("Fetching prover speed for address %s and duration %s", address, durationName)
			if err := fetchProverSpeed(address, duration, durationName); err != nil {
				log.Printf("Error fetching prover speed for address %s and duration %s: %v", address, durationName, err)
			}
		}

		log.Printf("Fetching prover reward for address %s", address)
		if err := fetchProverReward(address); err != nil {
			log.Printf("Error fetching prover reward for address %s: %v", address, err)
		}

		log.Printf("Fetching prover latest height for address %s", address)
		if err := fetchProverLatestHeight(address); err != nil {
			log.Printf("Error fetching prover latest height for address %s: %v", address, err)
		}
	}

	log.Println("Fetching latest block")
	if err := fetchLatestBlock(); err != nil {
		log.Printf("Error fetching latest block: %v", err)
	}
}

func fetchProverSpeed(address string, duration int, durationName string) error {
	url := fmt.Sprintf("%s/api/v1/provers/prover_speed_list", apiBaseURL)
	body := fmt.Sprintf(`{"address":["%s"],"duration":%d}`, address, duration)
	data, err := fetchData(url, body)
	if err != nil {
		return err
	}

	var result struct {
		Data struct {
			List []struct {
				Address string `json:"address"`
				Speed   string `json:"speed"`
			} `json:"list"`
			Total string `json:"total"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("could not unmarshal prover speed response: %w", err)
	}

	proverSpeed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "prover_speed",
		Help:        "Speed of the prover",
		ConstLabels: prometheus.Labels{"address": address, "duration": durationName},
	})
	prometheus.MustRegister(proverSpeed)

	for _, item := range result.Data.List {
		speed, err := parseFloat(item.Speed)
		if err != nil {
			log.Printf("Invalid speed value: %v", item.Speed)
			continue
		}
		proverSpeed.Set(speed)
		log.Printf("Set prover speed: address=%s, duration=%s, speed=%f", item.Address, durationName, speed)
	}

	totalSpeed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "total_speed",
		Help:        "Total speed",
		ConstLabels: prometheus.Labels{"duration": durationName},
	})
	prometheus.MustRegister(totalSpeed)

	totalSpeedValue, err := parseFloat(result.Data.Total)
	if err != nil {
		log.Printf("Invalid total speed value: %v", result.Data.Total)
		return nil
	}
	totalSpeed.Set(totalSpeedValue)
	log.Printf("Set total speed: duration=%s, total_speed=%f", durationName, totalSpeedValue)

	return push.New(pushgatewayURL, "prover_metrics").Collector(proverSpeed).Collector(totalSpeed).Push()
}

func fetchProverReward(address string) error {
	url := fmt.Sprintf("%s/api/v1/provers/prover_reward_list", apiBaseURL)
	body := fmt.Sprintf(`{"address":["%s"]}`, address)
	data, err := fetchData(url, body)
	if err != nil {
		return err
	}

	var result struct {
		Data struct {
			List []struct {
				Address     string `json:"address"`
				TotalReward string `json:"total_reward"`
			} `json:"list"`
			Total string `json:"total"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("could not unmarshal prover reward response: %w", err)
	}

	totalReward := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "total_reward",
		Help:        "Total reward of the prover",
		ConstLabels: prometheus.Labels{"address": address},
	})
	prometheus.MustRegister(totalReward)

	for _, item := range result.Data.List {
		totalRewardValue, err := parseFloat(item.TotalReward)
		if err != nil {
			log.Printf("Invalid total reward value: %v", item.TotalReward)
			continue
		}
		totalReward.Set(totalRewardValue)
		log.Printf("Set total reward: address=%s, total_reward=%f", item.Address, totalRewardValue)
	}

	return push.New(pushgatewayURL, "prover_metrics").Collector(totalReward).Push()
}

func fetchProverLatestHeight(address string) error {
	url := fmt.Sprintf("%s/api/v1/provers/prover_latest_height", apiBaseURL)
	body := fmt.Sprintf(`{"address":["%s"]}`, address)
	data, err := fetchData(url, body)
	if err != nil {
		return err
	}

	var result struct {
		Data []struct {
			Address string `json:"address"`
			Height  int    `json:"height"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("could not unmarshal prover latest height response: %w", err)
	}

	proverHeight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "prover_height",
		Help:        "Height of the prover",
		ConstLabels: prometheus.Labels{"address": address},
	})
	prometheus.MustRegister(proverHeight)

	for _, item := range result.Data {
		proverHeight.Set(float64(item.Height))
		log.Printf("Set prover height: address=%s, height=%d", item.Address, item.Height)
	}

	return push.New(pushgatewayURL, "prover_metrics").Collector(proverHeight).Push()
}

func fetchLatestBlock() error {
	url := fmt.Sprintf("%s/api/v1/chain/latest_block", apiBaseURL)
	data, err := fetchData(url, "")
	if err != nil {
		return err
	}

	log.Printf("Raw response from latest block: %s", string(data))

	var result struct {
		Data struct {
			Height         int    `json:"height"`
			ProofTarget    string `json:"proof_target"`
			CoinbaseReward string `json:"coinbase_reward"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("could not unmarshal latest block response: %w", err)
	}

	latestBlockHeight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "latest_block_height",
		Help: "Height of the latest block",
	})
	prometheus.MustRegister(latestBlockHeight)
	latestBlockHeight.Set(float64(result.Data.Height))
	log.Printf("Set latest block height: height=%d", result.Data.Height)

	proofTarget := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "proof_target",
		Help: "Proof target of the latest block",
	})
	prometheus.MustRegister(proofTarget)
	proofTargetValue, err := parseFloat(result.Data.ProofTarget)
	if err != nil {
		log.Printf("Invalid proof target value: %v", result.Data.ProofTarget)
		return nil
	}
	proofTarget.Set(proofTargetValue)
	log.Printf("Set proof target: proof_target=%f", proofTargetValue)

	coinbaseReward := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coinbase_reward",
		Help: "Coinbase reward of the latest block",
	})
	prometheus.MustRegister(coinbaseReward)
	coinbaseRewardValue, err := parseFloat(result.Data.CoinbaseReward)
	if err != nil {
		log.Printf("Invalid coinbase reward value: %v", result.Data.CoinbaseReward)
		return nil
	}
	coinbaseReward.Set(coinbaseRewardValue)
	log.Printf("Set coinbase reward: coinbase_reward=%f", coinbaseRewardValue)

	return push.New(pushgatewayURL, "latest_block_metrics").Collector(latestBlockHeight).Collector(proofTarget).Collector(coinbaseReward).Push()
}

func fetchData(url, body string) ([]byte, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch data: %w", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}

	log.Printf("Response from %s: %s", url, string(data))

	return data, nil
}

func toJSON(v interface{}) string {
	jsonData, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("Could not marshal to JSON: %v", err)
	}
	return string(jsonData)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
