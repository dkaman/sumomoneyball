package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"strings"
	"regexp"
	// "io/ioutil"


	"golang.org/x/net/html"
	"github.com/gosimple/slug"
)

const dateFormat = "January 2, 2006"

var RikishiURL string = "http://sumodb.sumogames.de/Rikishi.aspx"

// TODO: figure out what we wanna keep for basho results on this page...
// type BashoResult struct {
// 	Date string
// 	Rank string
// 	Wins int
// 	Losses int
// 	SumoDBBashoID int
// 	Prize string
// }

type Rikishi struct {
	SumoDBID    int
	HighestRank string
	RealName    string
	BirthDate   time.Time
	Origin      string
	Height      int
	Weight      int
	University  string
	Heya        string
	Shikona     string
	FirstBasho  string
}


func parseRikishiDataTable(node *html.Node, r *Rikishi) error {
	if node == nil {
		return fmt.Errorf("rikishiData table node is nil")
	}

	tbody := node.LastChild

	for row := tbody.FirstChild; row != nil; row = row.NextSibling {
		if row.FirstChild != nil {
			var keyText, valText string

			tdKey := row.FirstChild.NextSibling
			if tdKey != nil {
				keyTextNode := tdKey.FirstChild
				if keyTextNode != nil {
					keyText = keyTextNode.Data
				} else {
					continue
				}
			}

			tdVal := tdKey.NextSibling
			if tdVal != nil {
				valTextNode := tdVal.FirstChild
				if valTextNode != nil {
					valText = valTextNode.Data
				} else {
					continue
				}
			}

			switch keyText {
			case "Highest Rank":
				r.HighestRank = slug.Make(valText)

			case "Real Name":
				r.RealName = strings.ToLower(valText)

			case "Birth Date":
				re, err := regexp.Compile(`(\w+ \d+, \d{4}) \(\d+ years\)`)
				if err != nil {
					return err
				}

				result := re.FindStringSubmatch(valText)
				if len(result) != 2 {
					return fmt.Errorf("unable to match birthdate with provided regex. rikishi(%d)", r.SumoDBID)
				}

				birthDate, err := time.Parse(dateFormat, result[1])
				if err != nil {
					return err
				}

				r.BirthDate = birthDate

			case "Shusshin":
				r.Origin = strings.ToLower(valText)

			case "Height and Weight":
				re, err := regexp.Compile(`(\d+) cm (\d+) kg`)
				if err != nil {
					return err
				}

				result := re.FindStringSubmatch(valText)
				if len(result) != 3 {
					return fmt.Errorf("unable to match height and weight with provided regex. rikishi(%d)", r.SumoDBID)
				}

				height, err := strconv.Atoi(result[1])
				if err != nil {
					return err
				}

				weight, err := strconv.Atoi(result[2])
				if err != nil {
					return err
				}

				r.Height = height
				r.Weight = weight

			case "University":
				r.University = strings.ToLower(valText)

			case "Heya":
				r.Heya = strings.ToLower(valText)

			case "Shikona":
				r.Shikona = strings.ToLower(valText)

			case "Hatsu Dohyo":
				re, err := regexp.Compile(`(\d{4}\.\d{2}).*`)
				if err != nil {
					return err
				}

				result := re.FindStringSubmatch(valText)
				if len(result) != 2 {
					return fmt.Errorf("unable to match first basho with provided regex. rikishi(%d)", r.SumoDBID)
				}

				r.FirstBasho = result[1]
			}
		}
	}
	return nil
}

// TODO implement this one for the rikishi table
// func parseRikishiTable() (map[string]string, error){}

func getAttributeByName(node *html.Node, name string) (string, error) {
	for _, a := range node.Attr {
		if a.Key == name {
			return a.Val, nil
		}
	}

	return "", fmt.Errorf("no attribute named '%s' existed for that node.", name)
}

func parseHTMLResponse(doc *html.Node) (*Rikishi, error){
	var rikishiDataTable, rikishiTable *html.Node

	var crawler func(*html.Node) error

	crawler = func(node *html.Node) error{
		if node.Type == html.ElementNode && node.Data == "table" {
			a, err := getAttributeByName(node, "class")
			if err != nil {
				return err
			}

			switch a {
			case "rikishidata":
				if node.PrevSibling == nil && node.NextSibling == nil {
					// this is the inner rikishidata table
					rikishiDataTable = node
				}

			case "rikishi":
				rikishiTable = node
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
		return nil
	}

	err := crawler(doc)
	if err != nil {
		return nil, err
	}

	var r Rikishi

	err = parseRikishiDataTable(rikishiDataTable, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func Scrape(id int) (*Rikishi, error) {
	req, err := http.NewRequest("GET", RikishiURL, nil)

	q := url.Values{}
	q.Add("r", strconv.Itoa(id))

	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	rikishi, err := parseHTMLResponse(doc)
	if err != nil {
		return nil, err
	}

	rikishi.SumoDBID = id

	return rikishi, nil
}

func main() {
	// test on wakatakakage
	id := 12370
	rikishi, err := Scrape(id)
	if err != nil {
		fmt.Println("there was an issue scraping rikishi(%d): %s", id, err)
	}

	fmt.Printf("Shikona: %s (%d)\n", rikishi.Shikona, rikishi.SumoDBID)
	fmt.Printf("Real Name: %s\n", rikishi.RealName)
	fmt.Printf("Stable: %s\n", rikishi.Heya)
	fmt.Printf("Hometown: %s\n", rikishi.Origin)
	fmt.Printf("Birthdate: %s\n", rikishi.BirthDate.String())
	fmt.Printf("Highest Rank: %s\n", rikishi.HighestRank)
	fmt.Printf("University: %s\n", rikishi.University)
	fmt.Printf("Height (cm): %d\n", rikishi.Height)
	fmt.Printf("Weight (kg): %d\n", rikishi.Weight)
	fmt.Printf("First Basho: %s\n", rikishi.FirstBasho)
}
