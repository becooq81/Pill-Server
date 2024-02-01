package main

import (
	c "Pill-Server/config"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"

	"sync"

	"github.com/spf13/viper"
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

func main() {

	viper.SetConfigName("config")
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
	writer.Flush()

	file, err = os.Open("output.csv")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

}
