package prometh

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"log"
	"strconv"
)

func SpeedPush(url string, addr string, duration int, speed string) {
	job := "aleo_prover_speed"
	speedFloat, err := strconv.ParseFloat(speed, 64)
	if err != nil {
		log.Printf("parse speed %s failed:%s", speed, err)
		return
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: job})
	gauge.Set(speedFloat)
	err = push.New(url, job).Grouping("module", "cluster").Grouping("addr", addr).Grouping("duration", strconv.Itoa(duration)).Collector(gauge).Push()
	if err != nil {
		log.Printf("push prometheus %s failed:%s", url, err)
	}
}

func TotalSpeedPush(url string, duration int, speed string) {
	job := "aleo_prover_total_speed"
	speedFloat, err := strconv.ParseFloat(speed, 64)
	if err != nil {
		log.Printf("parse speed %s failed:%s", speed, err)
		return
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: job})
	gauge.Set(speedFloat)
	err = push.New(url, job).Grouping("duration", strconv.Itoa(duration)).Collector(gauge).Push()
	if err != nil {
		log.Printf("push prometheus %s failed:%s", url, err)
	}
}

func RewardPush(url string, addr string, reward string) {
	job := "aleo_prover_reward"
	rewardFloat, err := strconv.ParseFloat(reward, 64)
	if err != nil {
		log.Printf("parse reward %s failed:%s", reward, err)
		return
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: job})
	gauge.Set(rewardFloat)
	err = push.New(url, job).Grouping("module", "cluster").Grouping("addr", addr).Collector(gauge).Push()
	if err != nil {
		log.Printf("push prometheus %s failed:%s", url, err)
	}
}

func TotalRewardPush(url string, reward string) {
	job := "aleo_prover_total_reward"
	rewardFloat, err := strconv.ParseFloat(reward, 64)
	if err != nil {
		log.Printf("parse reward %s failed:%s", reward, err)
		return
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: job})
	gauge.Set(rewardFloat)
	err = push.New(url, job).Collector(gauge).Push()
	if err != nil {
		log.Printf("push prometheus %s failed:%s", url, err)
	}
}

func HeightPush(url string, addr string, height int) {
	job := "aleo_prover_latest_height"

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: job})
	gauge.Set(float64(height))
	err := push.New(url, job).Grouping("module", "cluster").Grouping("addr", addr).Collector(gauge).Push()
	if err != nil {
		log.Printf("push prometheus %s failed:%s", url, err)
	}
}

func BlockPush(url string, height int, proof string, reward string) {
	job := "aleo_prover_latest_block"

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: job})
	gauge.Set(float64(height))
	err := push.New(url, job).Grouping("type", "height").Collector(gauge).Push()
	if err != nil {
		log.Printf("push prometheus %s failed:%s", url, err)
	}

	proofFloat, err := strconv.ParseFloat(proof, 64)
	if err != nil {
		log.Printf("parse proof %s failed:%s", proof, err)
		return
	}
	gauge.Set(proofFloat)
	err = push.New(url, job).Grouping("type", "proof").Collector(gauge).Push()

	rewardFloat, err := strconv.ParseFloat(reward, 64)
	if err != nil {
		log.Printf("parse reward %s failed:%s", reward, err)
		return
	}
	gauge.Set(rewardFloat)
	err = push.New(url, job).Grouping("type", "reward").Collector(gauge).Push()

}
