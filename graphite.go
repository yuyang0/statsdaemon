package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"sort"
	"strconv"
	"time"
)

type graphiteBackend struct {
	addr string
}

func NewGraphiteBackend(addr string) backend {
	return &graphiteBackend{
		addr: addr,
	}
}

func (bd *graphiteBackend) submit(deadline time.Time) error {
	var buffer bytes.Buffer
	var num int64

	now := time.Now().Unix()

	client, err := net.Dial("tcp", bd.addr)
	if err != nil {
		if *debug {
			log.Printf("WARNING: resetting counters when in debug mode")
			processCounters(&buffer, now)
			processGauges(&buffer, now)
			processTimers(&buffer, now, percentThreshold)
			processSets(&buffer, now)
		}
		errmsg := fmt.Sprintf("dialing %s failed - %s", *graphiteAddress, err)
		return errors.New(errmsg)
	}
	defer client.Close()

	err = client.SetDeadline(deadline)
	if err != nil {
		return err
	}

	num += processCounters(&buffer, now)
	num += processGauges(&buffer, now)
	num += processTimers(&buffer, now, percentThreshold)
	num += processSets(&buffer, now)
	if num == 0 {
		return nil
	}

	if *debug {
		for _, line := range bytes.Split(buffer.Bytes(), []byte("\n")) {
			if len(line) == 0 {
				continue
			}
			log.Printf("DEBUG: %s", line)
		}
	}

	_, err = client.Write(buffer.Bytes())
	if err != nil {
		errmsg := fmt.Sprintf("failed to write stats - %s", err)
		return errors.New(errmsg)
	}

	log.Printf("sent %d stats to %s", num, *graphiteAddress)

	return nil
}

func processCounters(buffer *bytes.Buffer, now int64) int64 {
	var num int64
	// continue sending zeros for counters for a short period of time even if we have no new data
	for bucket, value := range counters {
		fmt.Fprintf(buffer, "%s %s %d\n", bucket, strconv.FormatFloat(value, 'f', -1, 64), now)
		delete(counters, bucket)
		countInactivity[bucket] = 0
		num++
	}
	for bucket, purgeCount := range countInactivity {
		if purgeCount > 0 {
			fmt.Fprintf(buffer, "%s 0 %d\n", bucket, now)
			num++
		}
		countInactivity[bucket] += 1
		if countInactivity[bucket] > *persistCountKeys {
			delete(countInactivity, bucket)
		}
	}
	return num
}

func processGauges(buffer *bytes.Buffer, now int64) int64 {
	var num int64

	for bucket, currentValue := range gauges {
		fmt.Fprintf(buffer, "%s %s %d\n", bucket, strconv.FormatFloat(currentValue, 'f', -1, 64), now)
		num++
		if *deleteGauges {
			delete(gauges, bucket)
		}
	}
	return num
}

func processSets(buffer *bytes.Buffer, now int64) int64 {
	num := int64(len(sets))
	for bucket, set := range sets {

		uniqueSet := map[string]bool{}
		for _, str := range set {
			uniqueSet[str] = true
		}

		fmt.Fprintf(buffer, "%s %d %d\n", bucket, len(uniqueSet), now)
		delete(sets, bucket)
	}
	return num
}

func processTimers(buffer *bytes.Buffer, now int64, pctls Percentiles) int64 {
	var num int64
	for bucket, timer := range timers {
		bucketWithoutPostfix := bucket[:len(bucket)-len(*postfix)]
		num++

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
					indexOfPerc -= 1 // index offset=0
				}
				maxAtThreshold = timer[indexOfPerc]
			}

			var tmpl string
			var pctstr string
			if pct.float >= 0 {
				tmpl = "%s.upper_%s%s %s %d\n"
				pctstr = pct.str
			} else {
				tmpl = "%s.lower_%s%s %s %d\n"
				pctstr = pct.str[1:]
			}
			threshold_s := strconv.FormatFloat(maxAtThreshold, 'f', -1, 64)
			fmt.Fprintf(buffer, tmpl, bucketWithoutPostfix, pctstr, *postfix, threshold_s, now)
		}

		mean_s := strconv.FormatFloat(mean, 'f', -1, 64)
		max_s := strconv.FormatFloat(max, 'f', -1, 64)
		min_s := strconv.FormatFloat(min, 'f', -1, 64)

		fmt.Fprintf(buffer, "%s.mean%s %s %d\n", bucketWithoutPostfix, *postfix, mean_s, now)
		fmt.Fprintf(buffer, "%s.upper%s %s %d\n", bucketWithoutPostfix, *postfix, max_s, now)
		fmt.Fprintf(buffer, "%s.lower%s %s %d\n", bucketWithoutPostfix, *postfix, min_s, now)
		fmt.Fprintf(buffer, "%s.count%s %d %d\n", bucketWithoutPostfix, *postfix, count, now)

		delete(timers, bucket)
	}
	return num
}
