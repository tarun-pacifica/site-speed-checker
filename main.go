package main

import (
	"fmt"
	"log"
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
	URL           string
	AvgLatency    time.Duration
	MinLatency    time.Duration
	MaxLatency    time.Duration
	FailureCount  int
	SuccessCount  int
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

	// Create a channel to limit concurrency
	semaphore := make(chan struct{}, concurrencyLimit)

	for _, site := range sites {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire a token
			defer func() { <-semaphore }() // Release the token

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
		"https://www.flashscore.com.au",
		"https://www.flashscore.com",
		"https://www.flashscore.co.uk",
		"https://www.flashscore.fr",
		"https://www.flashscore.es",
		"https://www.flashscore.de",
		"https://www.flashscore.com.br",
		"https://www.flashscore.ca",
		"https://www.flashscore.pl",
		"https://www.flashscore.nl",
		"https://www.flashscore.it",
		"https://www.flashscore.co.in",
		"https://www.flashscore.jp",
		"https://www.flashscore.kr",
		"https://www.flashscore.ru",
		"https://www.flashscore.mx",
	}

	runs := 100 // Number of times to repeat the test
	concurrencyLimit := len(validSites) // Maximum number of concurrent requests

	log.Printf("Starting latency tests with %d runs and concurrency limit of %d...\n", runs, concurrencyLimit)
	allResults := runLatencyTests(validSites, runs, concurrencyLimit)

	stats := calculateStats(allResults)

	fmt.Println("\nSummary of Flashscore sites latency (sorted by average latency):")
	sortedStats := make([]*SiteStats, 0, len(stats))
	for _, s := range stats {
		sortedStats = append(sortedStats, s)
	}
	sort.Slice(sortedStats, func(i, j int) bool {
		return sortedStats[i].AvgLatency < sortedStats[j].AvgLatency
	})

	for i, s := range sortedStats {
		fmt.Printf("%d. %s:\n", i+1, s.URL)
		fmt.Printf("   Avg: %v, Min: %v, Max: %v\n", s.AvgLatency, s.MinLatency, s.MaxLatency)
		fmt.Printf("   Success: %d, Failures: %d\n", s.SuccessCount, s.FailureCount)
	}
}