package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"sync"
)

var mutex = &sync.Mutex{}

// Response is the top-level structure for the XML response
type Response struct {
	Body Body `xml:"body"`
}

// Body represents the <body> section of the XML response
type Body struct {
	Items      Items `xml:"items"`
	NumOfRows  int   `xml:"numOfRows"`
	PageNo     int   `xml:"pageNo"`
	TotalCount int   `xml:"totalCount"`
}

// Items represents the <items> section inside <body>
type Items struct {
	Item []Item `xml:"item"`
}

// Item represents each <item> inside <items>
type Item struct {
	EntpName            string `xml:"entpName"`
	ItemName            string `xml:"itemName"`
	ItemSeq             string `xml:"itemSeq"`
	EfcyQesitm          string `xml:"efcyQesitm"`
	UseMethodQesitm     string `xml:"useMethodQesitm"`
	AtpnWarnQesitm      string `xml:"atpnWarnQesitm"`
	AtpnQesitm          string `xml:"atpnQesitm"`
	IntrcQesitm         string `xml:"intrcQesitm"`
	SeQesitm            string `xml:"seQesitm"`
	DepositMethodQesitm string `xml:"depositMethodQesitm"`
	OpenDe              string `xml:"openDe"`
	UpdateDe            string `xml:"updateDe"`
	ItemImage           string `xml:"itemImage"`
	Bizrno              string `xml:"bizrno"`
}

func removeTextWithinParentheses(text string) string {

	// Define a regular expression to match and remove text within parentheses, curly braces, and square brackets
	re := regexp.MustCompile(`\([^()]*\)|\{[^\{\}]*\}|\[[^\[\]]*\]`)

	// Apply the pattern iteratively until no more matches are found
	for re.MatchString(text) {
		text = re.ReplaceAllString(text, "")
	}

	// Remove extra spaces caused by the removal
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}

func processColumn(row []string) ([]string, []string) {
	processedText := removeTextWithinParentheses(row[1]) // Assuming '제품명' is the second column
	//fmt.Printf("Processed text: %s\n", processedText)
	processedText = regexp.MustCompile(`\|.*$`).ReplaceAllString(processedText, "")

	var splitTexts []string
	if !regexp.MustCompile(`\d+\.\d+%`).MatchString(processedText) {
		splitTexts = regexp.MustCompile(`\d+\.`).Split(processedText, -1)
		if len(splitTexts) > 1 {
			splitTexts = splitTexts[1:] // Remove the first element which is empty or not needed
		}
	}
	row[1] = processedText
	return row, splitTexts
}

func writePage(writer *csv.Writer, fullUrl string, page int) {
	// Send request to API
	fmt.Print("Fetching page ", page, "\n")
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("charset", "UTF-8")
	req.Header.Set("Authorization", "serviceKey")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	res, _ := io.ReadAll(resp.Body)

	var response Response
	err = xml.Unmarshal(res, &response)
	if err != nil {
		panic(err)
	}

	// Write item data to CSV
	mutex.Lock()
	for _, item := range response.Body.Items.Item {
		writer.Write([]string{item.EntpName, item.ItemName, item.ItemSeq, item.EfcyQesitm, item.UseMethodQesitm, item.AtpnWarnQesitm, item.AtpnQesitm, item.IntrcQesitm, item.SeQesitm, item.DepositMethodQesitm, item.OpenDe, item.UpdateDe, item.ItemImage, item.Bizrno})
	}
	mutex.Unlock()

}

func removeDuplicateRows(records [][]string) [][]string {
	// Create a map to store unique values from the second column
	uniqueSecondColumn := make(map[string]bool)

	// Create a new slice for the result without duplicates
	var result [][]string

	for _, row := range records {
		// Check if the value in the second column is non-empty and unique
		if row[1] != "" && !uniqueSecondColumn[row[1]] {
			uniqueSecondColumn[row[1]] = true
			result = append(result, row)
		}
	}

	return result
}

func main() {

	/* viper.SetConfigName("config")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	var configuration c.Configurations

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	// Set undefined variables
	viper.SetDefault("database.dbname", "test_db")
	err := viper.Unmarshal(&configuration)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
	}

	file, err := os.Create("output.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	writer.Write([]string{"entpName", "itemName", "itemSeq", "efcyQesitm", "useMethodQesitm", "atpnWarnQesitm", "atpnQesitm", "intrcQesitm", "seQesitm", "depositMethodQesitm", "openDe", "updateDe", "itemImage", "bizrno"})

	var totalCount, numOfRows int
	totalPages := 1

	var fullEndPoint = configuration.Api.EndPoint + "/" + configuration.Api.Operation + "?serviceKey=" + configuration.Api.ServiceKey
	req, err := http.NewRequest("GET", fullEndPoint, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("charset", "UTF-8")
	req.Header.Set("Authorization", "serviceKey")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	res, _ := io.ReadAll(resp.Body)

	var response Response
	err = xml.Unmarshal(res, &response)
	if err != nil {
		panic(err)
	}

	totalCount = response.Body.TotalCount
	numOfRows = response.Body.NumOfRows
	totalPages = (totalCount + 9) / numOfRows // Assuming 10 items per page
	fmt.Println("Total pages: ", totalPages)

	var wg sync.WaitGroup

	maxGoroutines := 10
	guard := make(chan struct{}, maxGoroutines)

	for page := 0; page <= totalPages; page++ {
		fullUrl := fullEndPoint + "&pageNo=" + fmt.Sprint(page)
		wg.Add(1)
		// Acquire a slot
		guard <- struct{}{}
		go func(fullUrl string, pageNo int) {
			defer wg.Done()
			// Your existing code for processing a page
			writePage(writer, fullUrl, pageNo)
			// Release the slot
			<-guard
		}(fullUrl, page)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	writer.Flush() */

	file, err := os.Open("output.csv")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()

	// print number of records
	fmt.Println(len(records))

	if err != nil {
		panic(err)
	}

	processedRows := [][]string{}
	for _, row := range records[1:] { // Skipping header row
		processedRow, splitTexts := processColumn(row)
		if len(splitTexts) > 0 {
			for _, splitText := range splitTexts {
				trimmedSplitText := strings.TrimSpace(splitText)
				splitTexts := strings.Split(trimmedSplitText, ",")
				for _, split := range splitTexts {
					newRow := make([]string, len(processedRow))
					copy(newRow, processedRow)
					newRow[1] = strings.TrimSpace(split)
					processedRows = append(processedRows, newRow)
				}
			}
		} else {
			processedRows = append(processedRows, processedRow)
		}
	}

	uniqueRecords := removeDuplicateRows(processedRows)

	outputFile, err := os.Create("processed_file.csv")
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	newWriter := csv.NewWriter(outputFile)
	defer newWriter.Flush()

	// Writing the header
	if len(records) > 0 {
		newWriter.Write(records[0]) // Assuming the first row is the header
	}

	for _, processedRow := range uniqueRecords {
		if err := newWriter.Write(processedRow); err != nil {
			panic(err)
		}
	}
	newWriter.Flush()

	rowNum := len(uniqueRecords)

	println(rowNum)

}
