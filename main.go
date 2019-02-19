// -----
// xkcd.go
//
// A program for indexing all existing xkcd comics, and allowing the CLI user to search them by keyword.
//
// Author: Kawai Washburn <kawaiwashburn@gmail.com>
// -----

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const xkcdURL = "https://xkcd.com/"
const xkcdSuffix = "info.0.json"
const indexPath = "./comix.dat"

type comic struct {
	Month      string
	Num        int
	Link       string
	Year       string
	News       string
	SafeTitle  string `json:"safe_title"`
	Transcript string
	Alt        string
	Img        string
	Title      string
	Day        string
}

type comicIdx struct {
	idx map[string]comic
}

func main() {
	args, err := getArgs()

	if err != nil {
		log.Fatal(err)
	}

	comIdx, err := loadIdx()
	if err != nil {
		log.Fatalf("Unable to load index from file: %s", err)
	}

	comm := args[0]
	switch comm {
	case "index":
		log.Printf("the current command is %s with last index %s\n", comm, args[1])
		// TODO: Break apart the current logic to determine the request index and the actual request
		getComics(args[1], &comIdx)
	case "search":
		log.Printf("the current command is %s with url %s\n", comm, args[1])
		phrase := args[1]

		cList, err := comIdx.search(phrase)
		if err != nil {
			log.Printf("Error while searching for comic: %s", err)
		}
		if len(cList) == 0 {
			log.Printf("No results found for '%s'", phrase)
		}

		// TODO: need to print the resulting cList in a more useful way
		for _, cmc := range cList {
			fmt.Printf("Found '%s' in comic %v, with transcript:\n \"%s\"\n\n", phrase, cmc.Num, cmc.Transcript)
		}
	}

	err = dumpIdx(&comIdx)
	if err != nil {
		log.Fatalf("Error writing index to disk: %v", err)
	}
}

func getArgs() ([]string, error) {
	args := os.Args[1:]

	if len(args) > 2 {
		return nil, errors.New("too many arguments supplied")
	} else if args[0] == "index" {
		if len(args) == 1 {
			args = append(args, "1")
		} else {
			if args[1] != "all" {
				i, err := strconv.Atoi(args[1])
				if err != nil {
					log.Fatalf("invalid value for index: %v", args[1])
				}
				if i < 0 {
					args[1] = "1"
				}
			}
		}
	}

	return args, nil
}

func loadIdx() (comicIdx, error) {
	cIdx := comicIdx{idx: make(map[string]comic)}
	b, err := ioutil.ReadFile(indexPath)
	if err != nil {
		return cIdx, errors.New("error opening index from disk at" + indexPath)
	}

	if err := json.Unmarshal(b, &cIdx.idx); err != nil {
		return cIdx, errors.New("error loading index from disk at" + indexPath)
	}

	return cIdx, nil
}

func dumpIdx(cIdx *comicIdx) error {
	idx, err := json.Marshal(cIdx.idx)
	if err != nil {
		return fmt.Errorf("Unable to encode comic index: %s", err)
	}

	err = ioutil.WriteFile(indexPath, idx, 0644)
	if err != nil {
		return fmt.Errorf("Unable to flush index to disk: %s", err)
	}

	return nil
}

// TODO: This is serial, and inefficient
func getComics(idx string, cIdx *comicIdx) {
	var n int

	// TODO: Take this section that calculates the index and place it in its own code
	latest, err := getLatest()
	if err != nil {
		log.Fatalf("Unable to get latest xkcd comic number: %s", err)
	}

	if idx != "all" {
		n, err = strconv.Atoi(idx)
		if err != nil {
			log.Fatalf("invalid value for index: %v", n)
		}
		n = n - 1
	} else {
		n = latest
	}
	// TODO: See note above

	for i := latest; i >= latest-n; i-- {
		log.Printf("Getting comic %v...", i)
		cNum := strconv.Itoa(n)

		if _, present := cIdx.idx[cNum]; !present {
			current, err := getXkcdComic(i)
			if err != nil {
				log.Printf("Unable to get xkcd comic: %s", err)
			} else {
				log.Printf("Got comic %v", current.Num)
				cIdx.indexComic(current)
			}
		} else {
			log.Printf("Comic already indexed.")
		}
	}
}

func getLatest() (int, error) {
	latest, err := getXkcdComic(0)
	if err != nil {
		return -1, err
	}

	log.Printf("Latest comic number is %v", latest.Num)
	return latest.Num, nil
}

func getXkcdComic(idx int) (comic, error) {
	var cNum string

	if idx == 0 {
		cNum = ""
	} else {
		cNum = strconv.Itoa(idx)
	}

	resp, err := http.Get(xkcdURL + cNum + "/" + xkcdSuffix)

	if err != nil {
		return comic{}, fmt.Errorf("could not get xkcd info from remote")
	} else if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return comic{}, fmt.Errorf("error in request to xcd: %s", resp.Status)
	}

	var current comic

	if err := json.NewDecoder(resp.Body).Decode(&current); err != nil {
		return comic{}, fmt.Errorf("unable to decode json value from xkcd: %s", err)
	}

	return current, nil
}

func (cIdx *comicIdx) indexComic(cmc comic) {
	cNum := strconv.Itoa(cmc.Num)

	if _, present := cIdx.idx[cNum]; !present {
		cIdx.idx[cNum] = cmc
		log.Printf("Indexed comic %v:", cNum)
	}
}

func (cIdx *comicIdx) search(phrase string) ([]comic, error) {
	var cList []comic

	for _, cmc := range cIdx.idx {
		if strings.Contains(cmc.Transcript, phrase) {
			cList = append(cList, cmc)
		}
	}

	return cList, nil
}
