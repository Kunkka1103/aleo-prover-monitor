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

	proverSpeed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prover_speed",
			Help: "Speed of the prover",
		},
		[]string{"address", "duration"},
	)
	totalSpeed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "total_speed",
			Help: "Total speed",
		},
		[]string{"duration"},
	)
	totalReward = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "total_reward",
			Help: "Total reward of the prover",
		},
		[]string{"address"},
	)
	proverHeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prover_height",
			Help: "Height of the prover",
		},
		[]string{"address"},
	)
	latestBlockHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "latest_block_height",
			Help: "Height of the latest block",
		},
	)
	proofTarget = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "proof_target",
			Help: "Proof target of the latest block",
		},
	)
	coinbaseReward = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "coinbase_reward",
			Help: "Coinbase reward of the latest block",
		},
	)
)

func init() {
	prometheus.MustRegister(proverSpeed)
	prometheus.MustRegister(totalSpeed)
	prometheus.MustRegister(totalReward)
	prometheus.MustRegister(proverHeight)
	prometheus.MustRegister(latestBlockHeight)
	prometheus.MustRegister(proofTarget)
	prometheus.MustRegister(coinbaseReward)
}

func main() {
	pflag.StringVar(&pushgatewayURL, "pushgateway", "http://pushgateway:9091", "URL of the Pushgateway")
	pflag.StringVar(&apiBaseURL, "api", "http://localhost:8088", "Base URL of the API")
	pflag.StringArrayVar(&addresses, "addresses", []string{
		"aleo1ul89ek6egwjtljy6yhmyteyu9y077ruahwggzfh6sgjqp890y5xs6mz9pe",
		"aleo1zp00ltnw23uvdq4spxax3zp84mt7pkvgyerlukxk5t443k6f5v9s9wem4l",
	}, "Addresses to monitor")

	pflag.DurationVar(&defaultDuration, "interval", 5*time.Minute, "Interval for fetching data")
	pflag.Parse()

	ticker := time.NewTicker(defaultDuration)
	defer ticker.Stop()

	log.Println("Starting prover monitor")

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

	for durationName, duration := range durations {
		log.Printf("Fetching prover speed for duration %s", durationName)
		fetchProverSpeed(addresses, duration, durationName)
	}

	log.Println("Fetching prover reward")
	fetchProverReward(addresses)

	log.Println("Fetching prover latest height")
	fetchProverLatestHeight(addresses)

	log.Println("Fetching latest block")
	fetchLatestBlock()

	log.Println("Pushing metrics to Pushgateway")
	if err := push.New(pushgatewayURL, "prover_metrics").
		Collector(proverSpeed).
		Collector(totalSpeed).
		Collector(totalReward).
		Collector(proverHeight).
		Collector(latestBlockHeight).
		Collector(proofTarget).
		Collector(coinbaseReward).
		Push(); err != nil {
		log.Fatalf("Could not push to Pushgateway: %v", err)
	}

	log.Println("Metrics pushed successfully")
}

func fetchProverSpeed(addresses []string, duration int, durationName string) {
	url := fmt.Sprintf("%s/api/v1/provers/prover_speed_list", apiBaseURL)
	body := fmt.Sprintf(`{"address":%v,"duration":%d}`, toJSON(addresses), duration)
	data := fetchData(url, body)

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
		log.Fatalf("Could not unmarshal prover speed response: %v", err)
	}

	for _, item := range result.Data.List {
		speed, err := parseFloat(item.Speed)
		if err != nil {
			log.Printf("Invalid speed value: %v", item.Speed)
			continue
		}
		proverSpeed.WithLabelValues(item.Address, durationName).Set(speed)
		log.Printf("Set prover speed: address=%s, duration=%s, speed=%f", item.Address, durationName, speed)
	}

	totalSpeedValue, err := parseFloat(result.Data.Total)
	if err != nil {
		log.Printf("Invalid total speed value: %v", result.Data.Total)
		return
	}
	totalSpeed.WithLabelValues(durationName).Set(totalSpeedValue)
	log.Printf("Set total speed: duration=%s, total_speed=%f", durationName, totalSpeedValue)
}

func fetchProverReward(addresses []string) {
	url := fmt.Sprintf("%s/api/v1/provers/prover_reward_list", apiBaseURL)
	body := fmt.Sprintf(`{"address":%v}`, toJSON(addresses))
	data := fetchData(url, body)

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
		log.Fatalf("Could not unmarshal prover reward response: %v", err)
	}

	for _, item := range result.Data.List {
		totalRewardValue, err := parseFloat(item.TotalReward)
		if err != nil {
			log.Printf("Invalid total reward value: %v", item.TotalReward)
			continue
		}
		totalReward.WithLabelValues(item.Address).Set(totalRewardValue)
		log.Printf("Set total reward: address=%s, total_reward=%f", item.Address, totalRewardValue)
	}
}

func fetchProverLatestHeight(addresses []string) {
	url := fmt.Sprintf("%s/api/v1/provers/prover_latest_height", apiBaseURL)
	body := fmt.Sprintf(`{"address":%v}`, toJSON(addresses))
	data := fetchData(url, body)

	var result struct {
		Data []struct {
			Address string `json:"address"`
			Height  int    `json:"height"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("Could not unmarshal prover latest height response: %v", err)
	}

	for _, item := range result.Data {
		proverHeight.WithLabelValues(item.Address).Set(float64(item.Height))
		log.Printf("Set prover height: address=%s, height=%d", item.Address, item.Height)
	}
}

func fetchLatestBlock() {
	url := fmt.Sprintf("%s/api/v1/chain/latest_block", apiBaseURL)
	data := fetchData(url, "")

	log.Printf("Raw response from latest block: %s", string(data))

	var result struct {
		Data struct {
			Height         int    `json:"height"`
			ProofTarget    string `json:"proof_target"`
			CoinbaseReward string `json:"coinbase_reward"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("Could not unmarshal latest block response: %v", err)
	}

	latestBlockHeight.Set(float64(result.Data.Height))
	log.Printf("Set latest block height: height=%d", result.Data.Height)

	proofTargetValue, err := parseFloat(result.Data.ProofTarget)
	if err != nil {
		log.Printf("Invalid proof target value: %v", result.Data.ProofTarget)
		return
	}
	proofTarget.Set(proofTargetValue)
	log.Printf("Set proof target: proof_target=%f", proofTargetValue)

	coinbaseRewardValue, err := parseFloat(result.Data.CoinbaseReward)
	if err != nil {
		log.Printf("Invalid coinbase reward value: %v", result.Data.CoinbaseReward)
		return
	}
	coinbaseReward.Set(coinbaseRewardValue)
	log.Printf("Set coinbase reward: coinbase_reward=%f", coinbaseRewardValue)
}

func fetchData(url, body string) []byte {
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		log.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Could not fetch data: %v", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Could not read response: %v", err)
	}

	log.Printf("Response from %s: %s", url, string(data))

	return data
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
