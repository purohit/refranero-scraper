package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var (
	printSlugs bool
	readSlugs  bool
)

type refran struct {
	idiom      string
	usage      string
	definition string
}

const (
	// Idioms start only with the below letters (http://cvc.cervantes.es/lengua/refranero/listado.aspx)
	letters = "ABCDEFGHIJLMNOPQRSTUVYZ"

	baseURL      = "http://cvc.cervantes.es/lengua/refranero"
	alphaPageURL = "http://cvc.cervantes.es/lengua/refranero/listado.aspx?letra="

	networkConns = 10

	sectionUsage      = "Marcador de uso:"
	sectionIdiom      = "Enunciado:"
	sectionDefinition = "Significado:"

	commonlyUsed = "Muy usado"
)

func main() {
	parseFlags()
	if printSlugs {
		outSlugs()
	} else if readSlugs {
		inSlugs()
	}
}

func parseFlags() {
	flag.BoolVar(&printSlugs, "print-slugs", false, "crawl all links for the entire alphabet, and print them line-by-line to stdout")
	flag.BoolVar(&readSlugs, "read-slugs", false, "read slugs, line by line, from stdin, and print the idiom, definition, etc if it is heavily used.")
	flag.Parse()
	if printSlugs && readSlugs {
		log.Fatal("Both -print-slugs and -read-slugs cannot be passed.")
	}
}

func inSlugs() {
	var done sync.WaitGroup
	output := make(chan refran, 0)
	jobs := make(chan string, 0)
	scanner := bufio.NewScanner(os.Stdin)
	for i := 0; i < networkConns; i++ {
		done.Add(1)
		go func() {
			for j := range jobs {
				doc, err := goquery.NewDocument(fmt.Sprintf("%s/%s", baseURL, j))
				if err != nil {
					log.Fatal(err)
				}
				sel := doc.Find("div.tabbertab").First()
				output <- refran{
					idiom:      getSectionText(sel, sectionIdiom),
					usage:      getSectionText(sel, sectionUsage),
					definition: getSectionText(sel, sectionDefinition),
				}
			}
			defer done.Done()
		}()
	}
	for scanner.Scan() {
		jobs <- scanner.Text()
	}
	close(jobs)
	go func() {
		done.Wait()
		close(output)
	}()
	fmt.Printf("Refran\tSignificado\n")
	for o := range output {
		if o.usage != commonlyUsed {
			continue
		}
		fmt.Printf("%s\t%s\n", o.idiom, o.definition)
	}
}

func getSectionText(sel *goquery.Selection, section string) string {
	child := sel.Find(fmt.Sprintf("p > strong:contains(\"%s\")", section))
	text := strings.TrimSpace(strings.TrimPrefix(child.Parent().Text(), section))
	return text
}

func outSlugs() {
	var wg sync.WaitGroup
	slugs := make(chan string, 0)
	for _, letter := range letters {
		wg.Add(1)
		go func(letter rune) {
			defer wg.Done()
			doc, err := goquery.NewDocument(fmt.Sprintf("%s%c", alphaPageURL, letter))
			if err != nil {
				log.Fatal(err)
			}

			doc.Find("ol#lista_az > li > a").Each(func(i int, s *goquery.Selection) {
				link, ok := s.Attr("href")
				if !ok {
					return
				}
				slugs <- link
			})
		}(letter)
	}
	go func() {
		wg.Wait()
		close(slugs)
	}()
	for slug := range slugs {
		fmt.Println(slug)
	}
}
