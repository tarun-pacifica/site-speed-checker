/*
* @Author: Tarun Mookhey
* @Date:   2024-10-06 17:38:17
* @Last Modified by:   Tarun Mookhey
* @Last Modified time: 2024-10-06 17:44:17
 */
package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

func main() {
	validSites := []string{
		"https://www.flashscore.com.au", // Australia
		"https://www.flashscore.com",    // International
		"https://www.flashscore.co.uk",  // United Kingdom
		"https://www.flashscore.fr",     // France
		"https://www.flashscore.es",     // Spain
		"https://www.flashscore.de",     // Germany
		"https://www.flashscore.com.br", // Brazil
		"https://www.flashscore.ca",     // Canada
		"https://www.flashscore.pl",     // Poland
		"https://www.flashscore.nl",     // Netherlands
		"https://www.flashscore.it",     // Italy
		"https://www.flashscore.co.in",  // India
		"https://www.flashscore.jp",     // Japan
		"https://www.flashscore.kr",     // South Korea
		"https://www.flashscore.ru",     // Russia
		"https://www.flashscore.mx",     // Mexico
		"https://www.flashscore.com.ar", // Argentina
		"https://www.flashscore.cl",     // Chile
		"https://www.flashscore.co",     // Colombia
		"https://www.flashscore.pe",     // Peru
		"https://www.flashscore.com.tr", // Turkey
		"https://www.flashscore.com.eg", // Egypt
		"https://www.flashscore.sa",     // Saudi Arabia
		"https://www.flashscore.ae",     // United Arab Emirates
		"https://www.flashscore.gr",     // Greece
		"https://www.flashscore.pt",     // Portugal
		"https://www.flashscore.be",     // Belgium
		"https://www.flashscore.se",     // Sweden
		"https://www.flashscore.no",     // Norway
		"https://www.flashscore.dk",     // Denmark
		"https://www.flashscore.fi",     // Finland
		"https://www.flashscore.cz",     // Czech Republic
		"https://www.flashscore.hu",     // Hungary
		"https://www.flashscore.ro",     // Romania
		"https://www.flashscore.bg",     // Bulgaria
		"https://www.flashscore.at",     // Austria
		"https://www.flashscore.ch",     // Switzerland
		"https://www.flashscore.ie",     // Ireland
		"https://www.flashscore.ua",     // Ukraine
		"https://www.flashscore.hr",     // Croatia
		"https://www.flashscore.rs",     // Serbia
		"https://www.flashscore.sk",     // Slovakia
		"https://www.flashscore.si",     // Slovenia
		"https://www.flashscore.lv",     // Latvia
		"https://www.flashscore.lt",     // Lithuania
		"https://www.flashscore.ee",     // Estonia
		"https://www.flashscore.com.my", // Malaysia
		"https://www.flashscore.com.sg", // Singapore
		"https://www.flashscore.com.ph", // Philippines
		"https://www.flashscore.co.th",  // Thailand
		"https://www.flashscore.co.id",  // Indonesia
		"https://www.flashscore.vn",     // Vietnam
		"https://www.flashscore.hk",     // Hong Kong
		"https://www.flashscore.tw",     // Taiwan
		"https://www.flashscore.co.nz",  // New Zealand
		"https://www.flashscore.co.za",  // South Africa
		"https://www.flashscore.com.ng", // Nigeria
		"https://www.flashscore.com.gh", // Ghana
		"https://www.flashscore.co.ke",  // Kenya
		"https://www.flashscore.com.tn", // Tunisia
		"https://www.flashscore.dz",     // Algeria
		"https://www.flashscore.ma",     // Morocco
		"https://www.flashscore.sn",     // Senegal
		"https://www.flashscore.ci",     // Ivory Coast
		"https://www.flashscore.cm",     // Cameroon
		"https://www.flashscore.com.uy", // Uruguay
		"https://www.flashscore.com.py", // Paraguay
		"https://www.flashscore.com.bo", // Bolivia
		"https://www.flashscore.com.ec", // Ecuador
		"https://www.flashscore.com.ve", // Venezuela
		"https://www.flashscore.com.co", // Colombia (alternative)
		"https://www.flashscore.com.gt", // Guatemala
		"https://www.flashscore.com.sv", // El Salvador
		"https://www.flashscore.hn",     // Honduras
		"https://www.flashscore.com.ni", // Nicaragua
		"https://www.flashscore.com.pa", // Panama
		"https://www.flashscore.com.do", // Dominican Republic
		"https://www.flashscore.com.cu", // Cuba
		"https://www.flashscore.com.pr", // Puerto Rico
		"https://www.flashscore.is",     // Iceland
		"https://www.flashscore.lu",     // Luxembourg
		"https://www.flashscore.mt",     // Malta
		"https://www.flashscore.cy",     // Cyprus
		"https://www.flashscore.by",     // Belarus
		"https://www.flashscore.md",     // Moldova
		"https://www.flashscore.am",     // Armenia
		"https://www.flashscore.ge",     // Georgia
		"https://www.flashscore.az",     // Azerbaijan
		"https://www.flashscore.kz",     // Kazakhstan
		"https://www.flashscore.uz",     // Uzbekistan
		"https://www.flashscore.kg",     // Kyrgyzstan
		"https://www.flashscore.tj",     // Tajikistan
		"https://www.flashscore.tm",     // Turkmenistan
		"https://www.flashscore.mn",     // Mongolia
		"https://www.flashscore.np",     // Nepal
		"https://www.flashscore.lk",     // Sri Lanka
		"https://www.flashscore.bd",     // Bangladesh
		"https://www.flashscore.pk",     // Pakistan
		"https://www.flashscore.af",     // Afghanistan
		"https://www.flashscore.ir",     // Iran
		"https://www.flashscore.iq",     // Iraq
		"https://www.flashscore.jo",     // Jordan
		"https://www.flashscore.lb",     // Lebanon
	}
	englishSites := detectEnglishSites(validSites)

	fmt.Println("Sites in English:")
	for _, site := range englishSites {
		fmt.Println(site)
	}
}

func detectEnglishSites(sites []string) []string {
	var wg sync.WaitGroup
	results := make(chan string, len(sites))

	for _, site := range sites {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if isEnglish(url) {
				results <- url
			}
		}(site)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var englishSites []string
	for site := range results {
		englishSites = append(englishSites, site)
	}

	return englishSites
}

func isEnglish(url string) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", url, err)
		return false
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading content from %s: %v\n", url, err)
		return false
	}

	doc, err := html.Parse(strings.NewReader(string(content)))
	if err != nil {
		fmt.Printf("Error parsing HTML from %s: %v\n", url, err)
		return false
	}

	lang := extractLanguage(doc)
	if lang == "" {
		lang = detectLanguage(content)
	}

	return strings.HasPrefix(lang, "en")
}

func extractLanguage(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "html" {
		for _, attr := range n.Attr {
			if attr.Key == "lang" {
				return attr.Val
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if lang := extractLanguage(c); lang != "" {
			return lang
		}
	}

	return ""
}

func detectLanguage(content []byte) string {
	langs := []language.Tag{
		language.English,
		language.French,
		language.German,
		language.Italian,
		language.Spanish,
		language.Portuguese,
		language.Russian,
		language.Japanese,
		language.Korean,
		// Add more languages as needed
	}

	matcher := language.NewMatcher(langs)
	tag, _ := language.MatchStrings(matcher, string(content))
	return display.English.Tags().Name(tag)
}
