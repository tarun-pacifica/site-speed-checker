package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

type SiteLatency struct {
	URL     string
	Latency time.Duration
}

type SiteStats struct {
	URL          string
	AvgLatency   time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
	FailureCount int
	SuccessCount int
}

func selectFlashscoreURL(stats []*SiteStats, thresholdPercent float64) string {
	if len(stats) < 2 {
		return stats[0].URL // Return the only (or first) URL if there are less than 2 sites
	}

	first := stats[0]
	second := stats[1]
	gap := second.AvgLatency - first.AvgLatency
	percentageDiff := float64(gap) / float64(first.AvgLatency) * 100

	if percentageDiff <= thresholdPercent {
		// If the difference is small, we still prefer the faster site,
		// but occasionally use the second to prevent overloading
		if rand.Float64() < 0.9 { // 90% chance to use the fastest
			return first.URL
		} else {
			return second.URL
		}
	} else {
		return first.URL // Always return the fastest if the gap is significant
	}
}

func measureLatency(url string) (time.Duration, error) {
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return time.Since(start), nil
}

func testLatency(sites []string, concurrencyLimit int) []SiteLatency {
	results := make([]SiteLatency, 0, len(sites))
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, concurrencyLimit)

	for _, site := range sites {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			latency, err := measureLatency(url)
			if err != nil {
				log.Printf("Error measuring latency for %s: %v", url, err)
				return
			}

			mu.Lock()
			results = append(results, SiteLatency{URL: url, Latency: latency})
			mu.Unlock()
		}(site)
	}

	wg.Wait()
	return results
}

func runLatencyTests(sites []string, runs, concurrencyLimit int) [][]SiteLatency {
	allResults := make([][]SiteLatency, runs)

	for i := 0; i < runs; i++ {
		log.Printf("Starting run %d of %d", i+1, runs)
		results := testLatency(sites, concurrencyLimit)
		allResults[i] = results

		sort.Slice(results, func(i, j int) bool {
			return results[i].Latency < results[j].Latency
		})

		log.Printf("Results for run %d:", i+1)
		for _, result := range results {
			log.Printf("%s: %v", result.URL, result.Latency)
		}
		log.Println()
	}

	return allResults
}

func calculateStats(allResults [][]SiteLatency) map[string]*SiteStats {
	stats := make(map[string]*SiteStats)

	for _, run := range allResults {
		for _, result := range run {
			if _, exists := stats[result.URL]; !exists {
				stats[result.URL] = &SiteStats{
					URL:        result.URL,
					MinLatency: result.Latency,
					MaxLatency: result.Latency,
				}
			}

			s := stats[result.URL]
			s.AvgLatency += result.Latency
			s.SuccessCount++

			if result.Latency < s.MinLatency {
				s.MinLatency = result.Latency
			}
			if result.Latency > s.MaxLatency {
				s.MaxLatency = result.Latency
			}
		}
	}

	totalRuns := len(allResults)
	for _, s := range stats {
		s.AvgLatency /= time.Duration(s.SuccessCount)
		s.FailureCount = totalRuns - s.SuccessCount
	}

	return stats
}

func main() {
validSites := []string{
	"https://www.flashscore.co.ke",
	"https://www.flashscore.co.za",
	"https://www.flashscore.com",
	"https://www.flashscore.com.au",
	"https://www.flashscore.com.ng",
	"https://www.flashscore.ca",
	"https://www.flashscore.am",
	"https://www.flashscore.hu",
	"https://www.flashscore.uz",
	"https://www.flashscore.ch",
	"https://www.flashscore.az",
	"https://www.flashscore.ie",
	"https://www.flashscore.co.in",
	"https://www.flashscore.ae",
	"https://www.flashscore.cz",
	"https://www.flashscore.lu",
	"https://www.flashscore.lv",
	"https://www.flashscore.com.bo",
	"https://www.flashscore.lt",
	"https://www.flashscore.is",
	"https://www.flashscore.be",
	"https://www.flashscore.co.uk",
	"https://www.flashscore.rs",
	"https://www.flashscore.af",
}
	runs := 10
	concurrencyLimit := len(validSites)

	log.Printf("Starting latency tests with %d runs and concurrency limit of %d...\n", runs, concurrencyLimit)
	allResults := runLatencyTests(validSites, runs, concurrencyLimit)

	stats := calculateStats(allResults)

	sortedStats := make([]*SiteStats, 0, len(stats))
	for _, s := range stats {
		sortedStats = append(sortedStats, s)
	}
	sort.Slice(sortedStats, func(i, j int) bool {
		return sortedStats[i].AvgLatency < sortedStats[j].AvgLatency
	})

	fmt.Println("\nSummary of Flashscore sites latency (sorted by average latency):")
	for i, s := range sortedStats {
		fmt.Printf("%d. %s:\n", i+1, s.URL)
		fmt.Printf("   Avg: %v, Min: %v, Max: %v\n", s.AvgLatency, s.MinLatency, s.MaxLatency)
		fmt.Printf("   Success: %d, Failures: %d\n", s.SuccessCount, s.FailureCount)

		if i < len(sortedStats)-1 {
			gap := sortedStats[i+1].AvgLatency - s.AvgLatency
			percentageDiff := float64(gap) / float64(s.AvgLatency) * 100
			fmt.Printf("   Gap to next: %v (%.2f%%)\n", gap, percentageDiff)
		}
		fmt.Println()
	}

	// Highlight the gap between first and second
	if len(sortedStats) >= 2 {
		first := sortedStats[0]
		second := sortedStats[1]
		gap := second.AvgLatency - first.AvgLatency
		percentageDiff := float64(gap) / float64(first.AvgLatency) * 100

		fmt.Println("======================================")
		fmt.Println("Gap between 1st and 2nd ranked sites:")
		fmt.Printf("1st: %s (Avg: %v)\n", first.URL, first.AvgLatency)
		fmt.Printf("2nd: %s (Avg: %v)\n", second.URL, second.AvgLatency)
		fmt.Printf("Absolute gap: %v\n", gap)
		fmt.Printf("Percentage difference: %.2f%%\n", percentageDiff)
		fmt.Println("======================================")
	}

	// Calculate and display overall statistics
	fastestSite := sortedStats[0]
	slowestSite := sortedStats[len(sortedStats)-1]
	totalGap := slowestSite.AvgLatency - fastestSite.AvgLatency
	averageGap := totalGap / time.Duration(len(sortedStats)-1)

	fmt.Println("\nOverall Statistics:")
	fmt.Printf("Total latency range: %v\n", totalGap)
	fmt.Printf("Average gap between sites: %v\n", averageGap)
	fmt.Printf("Percentage difference between fastest and slowest: %.2f%%\n",
		float64(totalGap)/float64(fastestSite.AvgLatency)*100)

	//
	fmt.Println("======================================")

	thresholdPercent := 2.0 // Set your desired threshold here
	selectedURL := selectFlashscoreURL(sortedStats, thresholdPercent)
	fmt.Printf("Selected URL: %s\n", selectedURL)
}
