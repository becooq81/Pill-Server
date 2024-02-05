package helloworld

import (
	"database/sql"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	_ "github.com/mattn/go-sqlite3"
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
type Config struct {
	EnvVars map[string]string `yaml:"environment_variables"`
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
	processedText = regexp.MustCompile(`\|.*$`).ReplaceAllString(processedText, "")
	var splitTexts []string
	if !regexp.MustCompile(`\d+\.\d+`).MatchString(processedText) {
		splitTexts = regexp.MustCompile(`\d+\.`).Split(processedText, -1)
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

// Temporary solution due to error in data provided by api.
func preprocessRecords(records [][]string) [][]string {
	var processedRecords [][]string
	for _, row := range records {
		if row[1] == "고려아시클로버크림(아시클로버)(수출명:바이락스크림(VIRAXCream)이노바이락스5%크림(INNOVIRAX5%Cream)" {
			row[1] = row[1] + ")"
		}
		processedRecords = append(processedRecords, row)
	}
	return processedRecords
}
func removeDuplicateAndSortRows(records [][]string) [][]string {
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
	// Sort the result by the second column value
	sort.SliceStable(result, func(i, j int) bool {
		return result[i][1] < result[j][1]
	})
	return result
}
func init() {
	functions.HTTP("HelloHTTP", helloHTTP)
}

// helloHTTP is an HTTP Cloud Function with a request parameter.
func helloHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// If not GET, return an error message and HTTP 405 Method Not Allowed status code
		http.Error(w, "Method is not supported.", http.StatusMethodNotAllowed)
		return
	}
	file, err := os.Create("output.csv")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Write CSV header
	writer.Write([]string{"entpName", "itemName", "itemSeq", "efcyQesitm", "useMethodQesitm", "atpnWarnQesitm", "atpnQesitm", "intrcQesitm", "seQesitm", "depositMethodQesitm", "openDe", "updateDe", "itemImage", "bizrno"})
	var totalCount, numOfRows int
	totalPages := 1
	var fullEndPoint = os.Getenv("URL") + os.Getenv("SERVICE") + "/" + os.Getenv("OPERATION") + "?serviceKey=" + os.Getenv("SERVICE_KEY")
	req, err := http.NewRequest("GET", fullEndPoint, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("charset", "UTF-8")
	req.Header.Set("Authorization", "serviceKey")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer resp.Body.Close()
	res, _ := io.ReadAll(resp.Body)
	var response Response
	err = xml.Unmarshal(res, &response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	totalCount = response.Body.TotalCount
	numOfRows = response.Body.NumOfRows
	totalPages = (totalCount + 9) / numOfRows // Assuming 10 items per page
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
	writer.Flush()
	file, err = os.Open("output.csv")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	preprocessedRecords := preprocessRecords(records[1:])
	processedRows := [][]string{}
	for _, row := range preprocessedRecords[1:] { // Including the header row
		if row[1] != "" { // Check if the first column is not empty
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
			} else if strings.Contains(processedRow[1], ",") {
				splitTexts := strings.Split(processedRow[1], ",")
				for _, split := range splitTexts {
					newRow := make([]string, len(processedRow))
					copy(newRow, processedRow)
					newRow[1] = strings.TrimSpace(split)
					processedRows = append(processedRows, newRow)
				}
			} else {
				processedRows = append(processedRows, processedRow)
			}
		}
	}
	uniqueRecords := removeDuplicateAndSortRows(processedRows)
	dbFileName := "processed_file.db"
	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	// Create table
	createTableSQL := `CREATE TABLE IF NOT EXISTS items (
		entpName TEXT, itemName TEXT PRIMARY KEY, itemSeq TEXT, efcyQesitm TEXT,
		useMethodQesitm TEXT, atpnWarnQesitm TEXT, atpnQesitm TEXT,
		intrcQesitm TEXT, seQesitm TEXT, depositMethodQesitm TEXT,
		openDe TEXT, updateDe TEXT, itemImage TEXT, bizrno TEXT);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	insertStmt := `INSERT INTO items (entpName, itemName, itemSeq, efcyQesitm, useMethodQesitm, atpnWarnQesitm, atpnQesitm, intrcQesitm, seQesitm, depositMethodQesitm, openDe, updateDe, itemImage, bizrno) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(insertStmt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer stmt.Close()
	for _, record := range uniqueRecords {
		if len(record) < 14 {
			// Replace with your error handling or log the issue
			fmt.Println("Record does not have enough fields")
			continue
		}
		_, err := stmt.Exec(record[0], record[1], record[2], record[3], record[4], record[5], record[6], record[7], record[8], record[9], record[10], record[11], record[12], record[13])
		if err != nil {
			// Replace with your error handling or log the issue
			fmt.Printf("Failed to insert record: %v\n", err)
		}
	}
	// Set the headers to indicate a db file is being returned
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\"processed_file.db\"")
	dbFile, err := os.Open(dbFileName)
	if err != nil {
		http.Error(w, "Failed to open the generated DB file.", http.StatusInternalServerError)
		return
	}
	defer dbFile.Close()
	// Copy the DB file content to the response writer
	_, err = io.Copy(w, dbFile)
	if err != nil {
		http.Error(w, "Failed to send the DB file.", http.StatusInternalServerError)
		return
	}
}
