package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	DEFAULT_ENDPOINT = "statsd"
)

type openFalconMsg struct {
	Metric      string  `json:"metric"`
	Endpoint    string  `json:"endpoint"`
	Tags        string  `json:"tags"`
	Value       float64 `json:"value"`
	Timestamp   int64   `json:"timestamp"`
	CounterType string  `json:"counterType"`
	Step        int64   `json:"step"`
}

type openFalconBackend struct {
	addr    string
	postUrl string
	packets []*openFalconMsg
}

func NewOpenFalconBackend(addr string) backend {
	return &openFalconBackend{
		addr:    addr,
		postUrl: fmt.Sprintf("http://%s/v1/push", addr),
	}
}

func (bd *openFalconBackend) appendPacket(metric string, value float64, now int64) {
	realMetric, tags := bd.parseMetric(metric)
	msg := &openFalconMsg{
		Metric:      realMetric,
		Endpoint:    DEFAULT_ENDPOINT,
		Tags:        tags,
		Value:       value,
		Timestamp:   now,
		CounterType: "GAUGE",
		Step:        *flushInterval,
	}
	bd.packets = append(bd.packets, msg)
}

func (bd *openFalconBackend) submit(deadline time.Time) error {
	var num int

	now := time.Now().Unix()

	defer func() {
		bd.packets = nil
	}()

	bd.processCounters(now)
	bd.processGauges(now)
	bd.processTimers(now, percentThreshold)
	bd.processSets(now)
	num = len(bd.packets)

	if num == 0 {
		return nil
	}

	buffer, err := json.Marshal(bd.packets)
	if err != nil {
		errmsg := fmt.Sprintf("failed to marshal json - %s", err)
		return errors.New(errmsg)
	}
	if *debug {
		log.Printf("DEBUG: %v", string(buffer))
	}
	req, err := http.NewRequest("POST", bd.postUrl, bytes.NewBuffer(buffer))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errmsg := fmt.Sprintf("failed to write stats - %s", err)
		return errors.New(errmsg)
	}
	defer resp.Body.Close()

	log.Printf("sent %d stats to %s", num, bd.addr)

	return nil
}

func (bd *openFalconBackend) processCounters(now int64) {
	// continue sending zeros for counters for a short period of time even if we have no new data
	for bucket, value := range counters {
		bd.appendPacket(bucket, value, now)

		delete(counters, bucket)
		countInactivity[bucket] = 0
	}
	for bucket, purgeCount := range countInactivity {
		if purgeCount > 0 {
			bd.appendPacket(bucket, 0, now)
		}
		countInactivity[bucket]++
		if countInactivity[bucket] > *persistCountKeys {
			delete(countInactivity, bucket)
		}
	}
}

func (bd *openFalconBackend) processGauges(now int64) {
	for bucket, currentValue := range gauges {
		bd.appendPacket(bucket, currentValue, now)

		if *deleteGauges {
			delete(gauges, bucket)
		}
	}
}

func (bd *openFalconBackend) processSets(now int64) {
	for bucket, set := range sets {

		uniqueSet := map[string]bool{}
		for _, str := range set {
			uniqueSet[str] = true
		}

		bd.appendPacket(bucket, float64(len(uniqueSet)), now)

		delete(sets, bucket)
	}
}

func (bd *openFalconBackend) processTimers(now int64, pctls Percentiles) {
	for bucket, timer := range timers {
		bucketWithoutPostfix := bucket[:len(bucket)-len(*postfix)]

		sort.Sort(timer)
		min := timer[0]
		max := timer[len(timer)-1]
		maxAtThreshold := max
		count := len(timer)

		sum := float64(0)
		for _, value := range timer {
			sum += value
		}
		mean := sum / float64(len(timer))

		for _, pct := range pctls {
			if len(timer) > 1 {
				var abs float64
				if pct.float >= 0 {
					abs = pct.float
				} else {
					abs = 100 + pct.float
				}
				// poor man's math.Round(x):
				// math.Floor(x + 0.5)
				indexOfPerc := int(math.Floor(((abs / 100.0) * float64(count)) + 0.5))
				if pct.float >= 0 {
					indexOfPerc-- // index offset=0
				}
				maxAtThreshold = timer[indexOfPerc]
			}

			var metric string
			var pctstr string
			if pct.float >= 0 {
				pctstr = pct.str
				metric = fmt.Sprintf("%s.upper_%s%s", bucketWithoutPostfix, pctstr, *postfix)
			} else {
				pctstr = pct.str[1:]
				metric = fmt.Sprintf("%s.lower_%s%s", bucketWithoutPostfix, pctstr, *postfix)
			}
			bd.appendPacket(metric, maxAtThreshold, now)
		}

		bd.appendPacket(fmt.Sprintf("%s.mean%s", bucketWithoutPostfix, *postfix), mean, now)
		bd.appendPacket(fmt.Sprintf("%s.upper%s", bucketWithoutPostfix, *postfix), max, now)
		bd.appendPacket(fmt.Sprintf("%s.lower%s", bucketWithoutPostfix, *postfix), min, now)
		bd.appendPacket(fmt.Sprintf("%s.count%s", bucketWithoutPostfix, *postfix), float64(count), now)

		delete(timers, bucket)
	}
}

func (bd *openFalconBackend) parseMetric(metric string) (string, string) {
	name := metric
	tags := ""
	index := strings.LastIndex(metric, "/")
	if index >= 0 {
		name = metric[:index]
		tags = metric[index+1:]
	}
	return name, tags
}
