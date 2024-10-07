package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type SiteMetrics struct {
	URL     string
	Latency time.Duration
	TTR     time.Duration
	Error   error
}

type SiteStats struct {
	URL          string
	AvgLatency   time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
	AvgTTR       time.Duration
	MinTTR       time.Duration
	MaxTTR       time.Duration
	FailureCount int
	SuccessCount int
}

type RankedSite struct {
	*SiteStats
	LatencyRank  int
	TTRRank      int
	CombinedRank float64
}

func measureMetrics(url string) (time.Duration, time.Duration, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var latency, ttr time.Duration
	start := time.Now()

	// Channel to signal when the page has visually completed loading
	visuallyComplete := make(chan bool, 1)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev.(type) {
		case *network.EventResponseReceived:
			if latency == 0 {
				latency = time.Since(start)
			}
		case *network.EventLoadingFinished:
			// This event might be too early for visual completion
			// We'll use a delay to approximate visual completion
			go func() {
				time.Sleep(500 * time.Millisecond)
				select {
				case visuallyComplete <- true:
				default:
				}
			}()
		}
	})

	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			select {
			case <-visuallyComplete:
				ttr = time.Since(start)
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}),
	)

	if err != nil {
		return 0, 0, err
	}

	return latency, ttr, nil
}

func testMetrics(sites []string, concurrencyLimit int) []SiteMetrics {
	results := make([]SiteMetrics, 0, len(sites))
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, concurrencyLimit)

	for _, site := range sites {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			latency, ttr, err := measureMetrics(url)
			mu.Lock()
			results = append(results, SiteMetrics{URL: url, Latency: latency, TTR: ttr, Error: err})
			mu.Unlock()
		}(site)
	}

	wg.Wait()
	return results
}

func runMetricsTests(sites []string, runs, concurrencyLimit int) [][]SiteMetrics {
	allResults := make([][]SiteMetrics, runs)

	for i := 0; i < runs; i++ {
		log.Printf("Starting run %d of %d", i+1, runs)
		results := testMetrics(sites, concurrencyLimit)
		allResults[i] = results

		log.Printf("Results for run %d:", i+1)
		for _, result := range results {
			if result.Error != nil {
				log.Printf("%s: Error: %v", result.URL, result.Error)
			} else {
				log.Printf("%s: Latency: %v, TTR: %v", result.URL, result.Latency, result.TTR)
			}
		}
		log.Println()

		if i < runs-1 {
			time.Sleep(3 * time.Second)
		}
	}

	return allResults
}

func calculateStats(allResults [][]SiteMetrics) map[string]*SiteStats {
	stats := make(map[string]*SiteStats)

	for _, run := range allResults {
		for _, result := range run {
			if _, exists := stats[result.URL]; !exists {
				stats[result.URL] = &SiteStats{
					URL:        result.URL,
					MinLatency: result.Latency,
					MaxLatency: result.Latency,
					MinTTR:     result.TTR,
					MaxTTR:     result.TTR,
				}
			}

			s := stats[result.URL]
			if result.Error == nil {
				s.AvgLatency += result.Latency
				s.AvgTTR += result.TTR
				s.SuccessCount++

				if result.Latency < s.MinLatency {
					s.MinLatency = result.Latency
				}
				if result.Latency > s.MaxLatency {
					s.MaxLatency = result.Latency
				}
				if result.TTR < s.MinTTR {
					s.MinTTR = result.TTR
				}
				if result.TTR > s.MaxTTR {
					s.MaxTTR = result.TTR
				}
			} else {
				s.FailureCount++
			}
		}
	}

	for _, s := range stats {
		if s.SuccessCount > 0 {
			s.AvgLatency /= time.Duration(s.SuccessCount)
			s.AvgTTR /= time.Duration(s.SuccessCount)
		}
	}

	return stats
}

func rankSites(stats map[string]*SiteStats) []RankedSite {
	sites := make([]RankedSite, 0, len(stats))
	for _, s := range stats {
		if s.SuccessCount > 0 {
			sites = append(sites, RankedSite{SiteStats: s})
		}
	}

	// Rank by Latency
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].AvgLatency == sites[j].AvgLatency {
			return sites[i].AvgTTR < sites[j].AvgTTR
		}
		return sites[i].AvgLatency < sites[j].AvgLatency
	})
	latencyRank := 1
	for i := range sites {
		if i > 0 && sites[i].AvgLatency != sites[i-1].AvgLatency {
			latencyRank = i + 1
		}
		sites[i].LatencyRank = latencyRank
	}

	// Rank by TTR
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].AvgTTR == sites[j].AvgTTR {
			return sites[i].AvgLatency < sites[j].AvgLatency
		}
		return sites[i].AvgTTR < sites[j].AvgTTR
	})
	ttrRank := 1
	for i := range sites {
		if i > 0 && sites[i].AvgTTR != sites[i-1].AvgTTR {
			ttrRank = i + 1
		}
		sites[i].TTRRank = ttrRank
	}

	// Calculate Combined Rank
	for i := range sites {
		sites[i].CombinedRank = float64(sites[i].LatencyRank+sites[i].TTRRank) / 2.0
	}

	// Sort by Combined Rank
	sort.Slice(sites, func(i, j int) bool {
		return sites[i].CombinedRank < sites[j].CombinedRank
	})

	return sites
}

func selectFlashscoreURL(rankedSites []RankedSite, thresholdPercent float64) string {
	if len(rankedSites) < 2 {
		return rankedSites[0].URL
	}

	first := rankedSites[0]
	second := rankedSites[1]
	gap := second.CombinedRank - first.CombinedRank
	percentageDiff := (gap / first.CombinedRank) * 100

	if percentageDiff <= thresholdPercent {
		if rand.Float64() < 0.9 {
			return first.URL
		} else {
			return second.URL
		}
	} else {
		return first.URL
	}
}

func main() {
	validSites := []string{
		"https://www.flashscore.co.ke",
		"https://www.flashscore.co.za",
		"https://www.flashscore.com",
		"https://www.flashscore.info",
		"https://www.flashscore.com.au",
		"https://www.flashscore.com.ng",
		"https://www.flashscore.ca",
		"https://www.flashscore.in",
		"https://www.flashscore.ae",
		"https://www.flashscore.co.uk"}
	runs := 3
	concurrencyLimit := len(validSites)

	log.Printf("Starting metrics tests with %d runs and concurrency limit of %d...\n", runs, concurrencyLimit)
	allResults := runMetricsTests(validSites, runs, concurrencyLimit)

	stats := calculateStats(allResults)
	rankedSites := rankSites(stats)

	// Display rankings by Latency
	fmt.Println("\nRankings by Latency:")
	sort.Slice(rankedSites, func(i, j int) bool {
		return rankedSites[i].LatencyRank < rankedSites[j].LatencyRank
	})
	for i, s := range rankedSites {
		fmt.Printf("%d. %s: Avg Latency: %v, Rank: %d\n", i+1, s.URL, s.AvgLatency, s.LatencyRank)
	}

	// Display rankings by TTR
	fmt.Println("\nRankings by TTR:")
	sort.Slice(rankedSites, func(i, j int) bool {
		return rankedSites[i].TTRRank < rankedSites[j].TTRRank
	})
	for i, s := range rankedSites {
		fmt.Printf("%d. %s: Avg TTR: %v, Rank: %d\n", i+1, s.URL, s.AvgTTR, s.TTRRank)
	}

	// Sort by Combined Rank for the final display
	sort.Slice(rankedSites, func(i, j int) bool {
		return rankedSites[i].CombinedRank < rankedSites[j].CombinedRank
	})

	fmt.Println("\nSummary of Flashscore sites metrics (sorted by combined rank):")
	for i, s := range rankedSites {
		fmt.Printf("%d. %s:\n", i+1, s.URL)
		fmt.Printf("   Avg Latency: %v, Min: %v, Max: %v\n", s.AvgLatency, s.MinLatency, s.MaxLatency)
		fmt.Printf("   Avg TTR: %v, Min: %v, Max: %v\n", s.AvgTTR, s.MinTTR, s.MaxTTR)
		fmt.Printf("   Latency Rank: %d, TTR Rank: %d, Combined Rank: %.2f\n", s.LatencyRank, s.TTRRank, s.CombinedRank)
		fmt.Printf("   Success: %d, Failures: %d\n", s.SuccessCount, s.FailureCount)

		if i < len(rankedSites)-1 {
			gap := rankedSites[i+1].CombinedRank - s.CombinedRank
			percentageDiff := (gap / s.CombinedRank) * 100
			fmt.Printf("   Gap to next: %.2f (%.2f%%)\n", gap, percentageDiff)
		}
		fmt.Println()
	}

	if len(rankedSites) >= 2 {
		first := rankedSites[0]
		second := rankedSites[1]
		gap := second.CombinedRank - first.CombinedRank
		percentageDiff := (gap / first.CombinedRank) * 100

		fmt.Println("======================================")
		fmt.Println("Gap between 1st and 2nd ranked sites:")
		fmt.Printf("1st: %s (Combined Rank: %.2f)\n", first.URL, first.CombinedRank)
		fmt.Printf("2nd: %s (Combined Rank: %.2f)\n", second.URL, second.CombinedRank)
		fmt.Printf("Absolute gap: %.2f\n", gap)
		fmt.Printf("Percentage difference: %.2f%%\n", percentageDiff)
		fmt.Println("======================================")
	}

	if len(rankedSites) > 0 {
		fastestSite := rankedSites[0]
		slowestSite := rankedSites[len(rankedSites)-1]
		totalGap := slowestSite.CombinedRank - fastestSite.CombinedRank
		averageGap := totalGap / float64(len(rankedSites)-1)

		fmt.Println("\nOverall Statistics:")
		fmt.Printf("Total Combined Rank range: %.2f\n", totalGap)
		fmt.Printf("Average gap between sites: %.2f\n", averageGap)
		fmt.Printf("Percentage difference between fastest and slowest: %.2f%%\n",
			(totalGap/fastestSite.CombinedRank)*100)

		fmt.Println("======================================")

		thresholdPercent := 2.0
		selectedURL := selectFlashscoreURL(rankedSites, thresholdPercent)
		fmt.Printf("Selected URL: %s\n", selectedURL)
	} else {
		fmt.Println("No successful measurements were made.")
	}
}
